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

// Package fn provides helpers for working with function properties.
package fn

import (
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

// GetCredentials returns credentials for the supplied key if they exist.
func GetCredentials(req *fnv1.RunFunctionRequest, key string) (map[string][]byte, error) {
	creds, err := request.GetCredentials(req, key)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve credential from Secret with name %q", key)
	}
	if creds.Type != resource.CredentialsTypeData {
		return nil, errors.Errorf("expected credential %q to be %q, got %q", key, resource.CredentialsTypeData, creds.Type)
	}
	return creds.Data, nil
}
