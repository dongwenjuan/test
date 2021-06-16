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
	"context"

	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
	"knative.dev/serving/pkg/apis/autoscaling"
	"knative.dev/serving/pkg/apis/serving"
)

// Validate validates the entire ActivationEndpoint.
func (aEP *ActivationEndpoint) Validate(ctx context.Context) *apis.FieldError {
	return serving.ValidateObjectMetadata(ctx, aEP.GetObjectMeta()).ViaField("metadata").
		Also(aEP.Spec.Validate(apis.WithinSpec(ctx)).ViaField("spec"))
}

// Validate validates ActivationEndpoint's Spec.
func (aEPs *ActivationEndpointSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(aEPs, &ActivationEndpointSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}
	return nil
}
