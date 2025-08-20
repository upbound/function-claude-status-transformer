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
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"

	"github.com/upbound/function-claude-status-transformer/input/v1beta1"
)

const (
	secretName = "aws"
	secretKey  = "credentials"
)

type AWS struct {
	valid bool
}

func New(in *v1beta1.StatusTransformation, req *fnv1.RunFunctionRequest) *AWS {
	if in.AWS == nil {
		return &AWS{}
	}

	_ = *in.AWS
	return &AWS{}
}

func (a *AWS) GetConfig() aws.Config {
	return aws.Config{}
}

func getRequestCredentials(req *fnv1.RunFunctionRequest) ([]byte, error) {
	c, err := request.GetCredentials(req, secretName)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get Anthropic API key from credential %q", secretName)
	}
	if c.Type != resource.CredentialsTypeData {
		return nil, errors.Errorf("expected credential %q to be %q, got %q", secretName, resource.CredentialsTypeData, c.Type)
	}
	b, ok := c.Data["credentials"]
	if !ok {
		return nil, errors.Errorf("credential %q is missing required key %q", secretName, secretKey)
	}

	return b, nil
}
