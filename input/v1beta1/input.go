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

	// AWS defines authentication and Bedrock configurations.
	// +optional
	// +kubebuilder:validation:Optional
	AWS *AWS `json:"aws"`
}

// AWS specifies configurations for working with AWS and ulimately Bedrock.
type AWS struct {
	// AWSBedrock provides configurations for working with AWS Bedrock as a
	// model provider.
	// +kubebuilder:validation:Required
	Bedrock Bedrock `json:"bedrock"`
	// Region specifies a specific region when this call is applicable.
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="us-east-1"
	Region string `json:"region"`
}

// Bedrock provides configurations for working with AWS Bedrock as a model
// provider.
type Bedrock struct {
	// ModelID is the Claude model to be used.
	// +kubebuilder:default="us.anthropic.claude-sonnet-4-20250514-v1:0"
	ModelID string `json:"modelID,omitempty"`
}

// UseAWS is a helper for determining if AWS configurations should be
// considered.
func (s *StatusTransformation) UseAWS() bool {
	return s.AWS != nil
}
