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

	"github.com/aws/aws-sdk-go-v2/aws"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/errors"

	"github.com/upbound/function-claude-status-transformer/input/v1alpha1"
	"github.com/upbound/function-claude-status-transformer/input/v1beta1"
	"github.com/upbound/function-claude-status-transformer/internal/credentials/aws/clients"
)

// AWS provides AWS specific credential access.
type AWS struct {
	c client.Client

	cfg *v1beta1.AWS
}

// New creates a new AWS.
func New(c client.Client, in *v1beta1.StatusTransformation) *AWS {
	a := &AWS{
		c:   c,
		cfg: in.AWS,
	}
	return a
}

// GetConfig returns an aws.Config derived from the request context and
// environment. Before attempting to construct the aws.Config, we pull
// the FunctionConfig from kube.
func (a *AWS) GetConfig(ctx context.Context) (*aws.Config, error) {
	fc := &v1alpha1.FunctionConfig{}
	if err := a.c.Get(ctx, types.NamespacedName{Name: a.cfg.FunctionConfigReference.Name}, fc); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve FunctionConfig")
	}
	return clients.GetAWSConfig(ctx, a.c, a.cfg.Region, fc)
}
