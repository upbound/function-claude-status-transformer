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

// Package anthropic provides credential helpers for working with Anthropic
// APIs.
package anthropic

import (
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

const (
	credName = "claude"
	credKey  = "ANTHROPIC_API_KEY"
)

// Anthropic provides API Key access.
type Anthropic struct {
	req *fnv1.RunFunctionRequest
}

// New constructs a new Anthropic.
func New(req *fnv1.RunFunctionRequest) *Anthropic {
	return &Anthropic{req: req}
}

// GetAPIKey retrieves the API key from the incoming RunFunctionRequest.
func (a *Anthropic) GetAPIKey() (string, error) {
	c, err := request.GetCredentials(a.req, credName)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get Anthropic API key from credential %q", credName)
	}
	if c.Type != resource.CredentialsTypeData {
		return "", errors.Errorf("expected credential %q to be %q, got %q", credName, resource.CredentialsTypeData, c.Type)
	}
	b, ok := c.Data[credKey]
	if !ok {
		return "", errors.Errorf("credential %q is missing required key %q", credName, credKey)
	}
	// TODO(negz): Where the heck is the newline at the end of this key
	// coming from? Bug in crossplane render?
	key := strings.Trim(string(b), "\n")

	return key, nil
}
