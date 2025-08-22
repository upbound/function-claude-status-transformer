/*
Copyright 2025 The Upbound Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main implements a Composition Function.
package main

import (
	"context"
	"log"
	"time"

	"github.com/alecthomas/kong"
	"golang.org/x/sync/errgroup"
	kruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/function-sdk-go"

	"github.com/upbound/function-claude-status-transformer/input/v1alpha1"
	"github.com/upbound/function-claude-status-transformer/internal/bootcheck"
)

func init() {
	err := bootcheck.CheckEnv()
	if err != nil {
		log.Fatalf("bootcheck failed. function will not be started: %v", err)
	}
	kruntime.Must(v1alpha1.AddToScheme(clientgoscheme.Scheme))
}

// CLI of this Function.
type CLI struct {
	Debug bool `short:"d" help:"Emit debug logs in addition to info logs."`

	Network            string `help:"Network on which to listen for gRPC connections." default:"tcp"`
	Address            string `help:"Address at which to listen for gRPC connections." default:":9443"`
	TLSCertsDir        string `help:"Directory containing server certs (tls.key, tls.crt) and the CA used to verify client certificates (ca.crt)" env:"TLS_SERVER_CERTS_DIR"`
	Insecure           bool   `help:"Run without mTLS credentials. If you supply this flag --tls-server-certs-dir will be ignored."`
	MaxRecvMessageSize int    `help:"Maximum size of received messages in MB." default:"4"`

	EnableFunctionConfigs bool `help:"Enable support for FunctionConfig APIs."`
}

// Run this Function.
func (c *CLI) Run() error {
	log, err := function.NewLogger(c.Debug)
	if err != nil {
		return err
	}
	g, ctx := errgroup.WithContext(context.Background())

	opts := []Option{}
	if c.EnableFunctionConfigs {
		cfg, err := ctrl.GetConfig()
		if err != nil {
			return errors.Wrap(err, "failed to get the kubeconfig for the FunctionConfig manager")
		}

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Cache: cache.Options{
				// TODO(tnthornton): expose the SyncInterval as a toggle if we
				// find this to be a feature we want to keep.
				SyncPeriod: ptr.To(time.Hour),
				ByObject: map[client.Object]cache.ByObject{
					&v1alpha1.FunctionConfig{}: {},
				},
			},
		})
		g.Go(func() error {
			err := mgr.Start(ctx)
			return errors.Wrap(ignoreCanceled(err), "failed to start manager")
		})

		opts = append(opts, WithClient(mgr.GetClient()))
	}

	return function.Serve(NewFunction(log, opts...),
		function.Listen(c.Network, c.Address),
		function.MTLSCertificates(c.TLSCertsDir),
		function.Insecure(c.Insecure),
		function.MaxRecvMessageSize(c.MaxRecvMessageSize*1024*1024))
}

func main() {
	ctx := kong.Parse(&CLI{}, kong.Description("A Crossplane Composition Function."))
	ctx.FatalIfErrorf(ctx.Run())
}

func ignoreCanceled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
