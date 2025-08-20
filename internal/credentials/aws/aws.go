// /*
// Copyright 2025 The Upbound Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package aws provides credential helpers for working with AWS Bedrock APIs.
package aws

import (
	"context"
	"maps"

	"github.com/aws/aws-sdk-go-v2/aws"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"

	"github.com/upbound/function-claude-status-transformer/input/v1alpha1"
	"github.com/upbound/function-claude-status-transformer/input/v1beta1"
	"github.com/upbound/function-claude-status-transformer/internal/credentials/aws/clients"
	"github.com/upbound/function-claude-status-transformer/internal/credentials/fn"
)

// AWS provides AWS specific credential access.
type AWS struct {
	c client.Client

	cfg *v1beta1.AWS
	req *fnv1.RunFunctionRequest
}

// New creates a new AWS.
func New(in *v1beta1.StatusTransformation, req *fnv1.RunFunctionRequest) *AWS {
	a := &AWS{
		c:   &fnCredClient{req: req},
		cfg: in.AWS,
		req: req,
	}
	return a
}

// GetConfig returns an aws.Config derived from the request context and
// environment.
func (a *AWS) GetConfig(ctx context.Context) (*aws.Config, error) {
	data, err := fn.GetCredentials(a.req, "function-config")
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve function-config")
	}

	fc := &v1alpha1.FunctionConfig{}
	if err := yaml.Unmarshal(data["config.yaml"], fc); err != nil {
		return nil, errors.Wrap(err, "invalid FunctionConfig found in secret")
	}

	return clients.GetAWSConfig(ctx, a.c, a.cfg.Region, fc)
}

var _ client.Client = &fnCredClient{}

// fnCredClient is a simple client that embeds the upstream test.MockClient in
// order to statisfy the client.Client interface contract. It has only one
// purpose, that is to redirect requests to Secrets back to the incoming
// Function Request so that we do not need to provide access to the API server.
type fnCredClient struct {
	req *fnv1.RunFunctionRequest

	test.MockClient
}

// Get mocks the standard client.Client get call to pull the Secret retrieval
// from the incoming function request, rather than going to kube. This enables
// us to utilize some helpers from c/crossplane-runtime without needing to give
// the function a client with API server access.
func (c *fnCredClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	s, ok := obj.(*corev1.Secret)
	if !ok {
		return errors.New("invalid object Kind supplied for retrieval, should be Secret but was not")
	}

	s.SetName(key.Name)
	s.SetNamespace(key.Namespace)

	data, err := fn.GetCredentials(c.req, key.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve credentials for %q", key.Name)
	}

	// copy data from the creds map to the secret
	maps.Copy(s.Data, data)

	return nil
}
