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

package activationendpoint

import (
	"context"
	"errors"

	"knative.dev/serving/pkg/apis/autoscaling/v1alpha1"

	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"

    corev1listers "k8s.io/client-go/listers/core/v1"
	listers "knative.dev/serving/pkg/client/listers/serving/v1"
	revisioninformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/revision"
	aepreconciler "knative.dev/serving/pkg/client/injection/reconciler/autoscaling/v1alpha1/activationendpoint"

)

// reconciler implements controller.Reconciler for ActivationEndpoint resources.
type reconciler struct {
	kubeclient kubernetes.Interface

	// listers index properties about resources
	endpointsLister corev1listers.EndpointsLister
	revisionLister  listers.RevisionLister
}

// Check that our Reconciler implements metricreconciler.Interface
var _ aepreconciler.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(_ context.Context, ActivationEndpoint *v1alpha1.ActivationEndpoint) pkgreconciler.Event {

	activatorEps, err := r.endpointsLister.Endpoints(system.Namespace()).Get(networking.ActivatorServiceName)
	if err != nil {
		return fmt.Errorf("failed to get activator service endpoints: %w", err)
	}

    ActivationEpNum = subsetActivationEndpointsNum(activatorEps)
    subEps = subsetActivationEndpoints(activatorEps, ActivationEpNum)

	ActivationEndpoint.Status.MarkActivationEndpointReady()

	return nil
}

func subsetActivationEndpointsNum(eps *corev1.Endpoints) int {

    return min(len(eps), 3)
}

func subsetActivationEndpoints(eps *corev1.Endpoints, target string, n int) *corev1.Endpoints {
	// n == 0 means all, and if there are no subsets there's no work to do either.
	if len(eps.Subsets) == 0 || n == 0 {
		return eps
	}

	addrs := make(sets.String, len(eps.Subsets[0].Addresses))
	for _, ss := range eps.Subsets {
		for _, addr := range ss.Addresses {
			addrs.Insert(addr.IP)
		}
	}

	// The input is not larger than desired.
	if len(addrs) <= n {
		return eps
	}

	selection := hash.ChooseSubset(addrs, n, target)

	// Copy the informer's copy, so we can filter it out.
	neps := eps.DeepCopy()
	// Standard in place filter using read and write indices.
	// This preserves the original object order.
	r, w := 0, 0
	for r < len(neps.Subsets) {
		ss := neps.Subsets[r]
		// And same algorithm internally.
		ra, wa := 0, 0
		for ra < len(ss.Addresses) {
			if selection.Has(ss.Addresses[ra].IP) {
				ss.Addresses[wa] = ss.Addresses[ra]
				wa++
			}
			ra++
		}
		// At least one address from the subset was preserved, so keep it.
		if wa > 0 {
			ss.Addresses = ss.Addresses[:wa]
			// At least one address from the subset was preserved, so keep it.
			neps.Subsets[w] = ss
			w++
		}
		r++
	}
	// We are guaranteed here to have w > 0, because
	// 0. There's at least one subset (checked above).
	// 1. A subset cannot be empty (k8s validation).
	// 2. len(addrs) is at least as big as n
	// Thus there's at least 1 non empty subset (and for all intents and purposes we'll have 1 always).
	neps.Subsets = neps.Subsets[:w]
	return neps
}