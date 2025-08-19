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

// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=function-claude-status-transformer.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusTransformation can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type StatusTransformation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AdditionalContext is additional context that the user may provide to help
	// Claude identify the issue.
	AdditionalContext string `json:"additionalContext"`

	// AWSBedrock provides configurations for working with AWS Bedrock as a
	// model provider.
	// +optional
	AWSBedrock *AWSBedrock `json:"bedrock"`
}

// AWSBedrock provides configurations for working with AWS Bedrock as a model
// provider.
type AWSBedrock struct {
	// ModelID is the Claude model to be used.
	// +kubebuilder:default="us.anthropic.claude-sonnet-4-20250514-v1:0"
	ModelID string `json:"modelID,omitempty"`
	// UseFnCredentials indicates whether the function should use the
	// credentials passed from the function request or not. Default is false.
	UseFnCredentials bool `json:"useFnCredentials"`
}
