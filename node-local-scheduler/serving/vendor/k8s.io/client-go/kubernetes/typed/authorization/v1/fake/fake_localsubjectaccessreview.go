/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	testing "k8s.io/client-go/testing"
)

// FakeLocalSubjectAccessReviews implements LocalSubjectAccessReviewInterface
type FakeLocalSubjectAccessReviews struct {
	Fake *FakeAuthorizationV1
	ns   string
}

var localsubjectaccessreviewsResource = schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "localsubjectaccessreviews"}

var localsubjectaccessreviewsKind = schema.GroupVersionKind{Group: "authorization.k8s.io", Version: "v1", Kind: "LocalSubjectAccessReview"}

// Create takes the representation of a localSubjectAccessReview and creates it.  Returns the server's representation of the localSubjectAccessReview, and an error, if there is any.
func (c *FakeLocalSubjectAccessReviews) Create(ctx context.Context, localSubjectAccessReview *v1.LocalSubjectAccessReview, opts metav1.CreateOptions) (result *v1.LocalSubjectAccessReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(localsubjectaccessreviewsResource, c.ns, localSubjectAccessReview), &v1.LocalSubjectAccessReview{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.LocalSubjectAccessReview), err
}
