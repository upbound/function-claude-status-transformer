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
}
