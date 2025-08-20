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

package aws

import (
	"context"
	"maps"

	"github.com/aws/aws-sdk-go-v2/aws"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"

	"github.com/upbound/function-claude-status-transformer/input/v1beta1"
	"github.com/upbound/function-claude-status-transformer/internal/credentials/aws/clients"
)

// AWS provides AWS specific credential access.
type AWS struct {
	c client.Client

	cfg *v1beta1.AWS
}

// New creates a new AWS.
func New(in *v1beta1.StatusTransformation, req *fnv1.RunFunctionRequest) *AWS {
	a := &AWS{
		c:   &fnCredClient{req: req},
		cfg: in.AWS,
	}
	return a
}

// GetConfig returns an aws.Config derived from the request context and
// environment.
func (a *AWS) GetConfig(ctx context.Context) (*aws.Config, error) {
	return clients.GetAWSConfig(ctx, a.c, a.cfg)
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
func (c *fnCredClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	s, ok := obj.(*corev1.Secret)
	if !ok {
		return errors.New("invalid object Kind supplied for retrieval, should be Secret but was not")
	}

	s.SetName(key.Name)
	s.SetNamespace(key.Namespace)

	creds, err := request.GetCredentials(c.req, key.Name)
	if err != nil {
		return errors.Wrapf(err, "cannot retrieve credential from Secret with name %q", key.Name)
	}
	if creds.Type != resource.CredentialsTypeData {
		return errors.Errorf("expected credential %q to be %q, got %q", key.Name, resource.CredentialsTypeData, creds.Type)
	}

	// copy data from the creds map to the secret
	maps.Copy(s.Data, creds.Data)

	return nil
}
