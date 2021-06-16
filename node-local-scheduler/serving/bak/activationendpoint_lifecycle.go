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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

const (
	// ActivationEndpointConditionReady is set when the Metric's latest
	// underlying revision has reported readiness.
	ActivationEndpointConditionReady = apis.ConditionReady
)

var condSet = apis.NewLivingConditionSet(
	ActivationEndpointConditionReady,
)

// GetConditionSet retrieves the condition set for this resource.
// Implements the KRShaped interface.
func (*ActivationEndpoint) GetConditionSet() apis.ConditionSet {
	return condSet
}

// GetGroupVersionKind implements OwnerRefable.
func (m *ActivationEndpoint) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ActivationEndpoint")
}

// GetCondition gets the condition `t`.
func (aEPs *ActivationEndpointStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return condSet.Manage(aEPs).GetCondition(t)
}

// InitializeConditions initializes the conditions of the Metric.
func (aEPs *ActivationEndpointStatus) InitializeConditions() {
	condSet.Manage(aEPs).InitializeConditions()
}

// MarkActivationEndpointReady marks the ActivationEndpoint status as ready
func (aEPs *ActivationEndpointStatus) MarkActivationEndpointReady() {
	condSet.Manage(aEPs).MarkTrue(ActivationEndpointConditionReady)
}

// MarkActivationEndpointNotReady marks the ActivationEndpoint status as ready == Unknown
func (aEPs *ActivationEndpointStatus) MarkActivationEndpointNotReady(reason, message string) {
	condSet.Manage(aEPs).MarkUnknown(ActivationEndpointConditionReady, reason, message)
}

// MarkActivationEndpointFailed marks the ActivationEndpoint status as failed
func (aEPs *ActivationEndpointStatus) MarkActivationEndpointFailed(reason, message string) {
	condSet.Manage(aEPs).MarkFalse(ActivationEndpointConditionReady, reason, message)
}

// IsReady returns true if the Status condition ActivationEndpointConditionReady
// is true and the latest spec has been observed.
func (aEP *ActivationEndpoint) IsReady() bool {
	aEPs := aEP.Status
	return aEPs.ObservedGeneration == aEP.Generation &&
		aEPs.GetCondition(ActivationEndpointConditionReady).IsTrue()
}
