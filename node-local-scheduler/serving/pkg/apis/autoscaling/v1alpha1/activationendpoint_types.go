/*
Copyright 2019 The Knative Authors

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

package v1alpha1

import (
	"time"

    corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/autoscaling"
)

// ActivationEndpoint represents a subset of avtiovator endpoints.
//
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActivationEndpoint struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Metric (from the client).
	// +optional
	Spec ActivationEndpointSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Metric (from the controller).
	// +optional
	Status ActivationEndpointStatus `json:"status,omitempty"`
}

// Verify that Metric adheres to the appropriate interfaces.
var (
	// Check that Metric can be validated and can be defaulted.
	_ apis.Validatable = (*ActivationEndpoint)(nil)
	_ apis.Defaultable = (*ActivationEndpoint)(nil)

	// Check that we can create OwnerReferences to a ActivationEndpoint.
	_ kmeta.OwnerRefable = (*ActivationEndpoint)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*ActivationEndpoint)(nil)
)

// ActivationEndpointSpec contains all values a ActivationEndpoint needs to operate.
type ActivationEndpointSpec struct {
	// ScrapeTarget is the K8s service that publishes the metric endpoint.
	Reachability string `json:"reachability,omitempty"`
}

// ActivationEndpointStatus reflects the status of ActivationEndpoint for this specific entity.
type ActivationEndpointStatus struct {
	duckv1.Status                       `json:",inline"`
	desiredActivationEpNum  int         `json:"desiredactivationEpNum,omitempty"`
	actualActivationEpNum  int          `json:"actualactivationEpNum,omitempty"`
	Subsets []corev1.Endpoints          `json:"subsets,omitempty"`
}

// MetricList is a list of Metric resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActivationEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ActivationEndpoint `json:"items"`
}

// GetStatus retrieves the status of the ActivationEndpoint. Implements the KRShaped interface.
func (activationEP *ActivationEndpoint) GetStatus() *duckv1.Status {
	return &activationEP.Status.Status
}
