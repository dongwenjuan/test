/*
Copyright 2020 The Knative Authors

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

package v1alpha1

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	scheme "knative.dev/serving/pkg/client/clientset/versioned/scheme"
)

// ActivationEndpointsGetter has a method to return a ActivationEndpointInterface.
// A group's client should implement this interface.
type ActivationEndpointsGetter interface {
	ActivationEndpoints(namespace string) ActivationEndpointInterface
}

// ActivationEndpointInterface has methods to work with ActivationEndpoint resources.
type ActivationEndpointInterface interface {
	Create(ctx context.Context, activationEndpoint *v1alpha1.ActivationEndpoint, opts v1.CreateOptions) (*v1alpha1.ActivationEndpoint, error)
	Update(ctx context.Context, activationEndpoint *v1alpha1.ActivationEndpoint, opts v1.UpdateOptions) (*v1alpha1.ActivationEndpoint, error)
	UpdateStatus(ctx context.Context, activationEndpoint *v1alpha1.ActivationEndpoint, opts v1.UpdateOptions) (*v1alpha1.ActivationEndpoint, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ActivationEndpoint, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ActivationEndpointList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ActivationEndpoint, err error)
	ActivationEndpointExpansion
}

// activationEndpoints implements ActivationEndpointInterface
type activationEndpoints struct {
	client rest.Interface
	ns     string
}

// newActivationEndpoints returns a ActivationEndpoints
func newActivationEndpoints(c *AutoscalingV1alpha1Client, namespace string) *activationEndpoints {
	return &activationEndpoints{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the activationEndpoint, and returns the corresponding activationEndpoint object, and an error if there is any.
func (c *activationEndpoints) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ActivationEndpoint, err error) {
	result = &v1alpha1.ActivationEndpoint{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("activationendpoints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ActivationEndpoints that match those selectors.
func (c *activationEndpoints) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ActivationEndpointList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ActivationEndpointList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("activationendpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested activationEndpoints.
func (c *activationEndpoints) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("activationendpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a activationEndpoint and creates it.  Returns the server's representation of the activationEndpoint, and an error, if there is any.
func (c *activationEndpoints) Create(ctx context.Context, activationEndpoint *v1alpha1.ActivationEndpoint, opts v1.CreateOptions) (result *v1alpha1.ActivationEndpoint, err error) {
	result = &v1alpha1.ActivationEndpoint{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("activationendpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(activationEndpoint).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a activationEndpoint and updates it. Returns the server's representation of the activationEndpoint, and an error, if there is any.
func (c *activationEndpoints) Update(ctx context.Context, activationEndpoint *v1alpha1.ActivationEndpoint, opts v1.UpdateOptions) (result *v1alpha1.ActivationEndpoint, err error) {
	result = &v1alpha1.ActivationEndpoint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("activationendpoints").
		Name(activationEndpoint.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(activationEndpoint).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *activationEndpoints) UpdateStatus(ctx context.Context, activationEndpoint *v1alpha1.ActivationEndpoint, opts v1.UpdateOptions) (result *v1alpha1.ActivationEndpoint, err error) {
	result = &v1alpha1.ActivationEndpoint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("activationendpoints").
		Name(activationEndpoint.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(activationEndpoint).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the activationEndpoint and deletes it. Returns an error if one occurs.
func (c *activationEndpoints) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("activationendpoints").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *activationEndpoints) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("activationendpoints").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched activationEndpoint.
func (c *activationEndpoints) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ActivationEndpoint, err error) {
	result = &v1alpha1.ActivationEndpoint{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("activationendpoints").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
