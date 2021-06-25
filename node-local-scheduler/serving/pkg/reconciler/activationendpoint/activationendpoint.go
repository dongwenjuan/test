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


type RevIDSet map[String][]types.NamespacedName

// reconciler implements controller.Reconciler for ActivationEndpoint resources.
type reconciler struct {
	kubeclient kubernetes.Interface

	// listers index properties about resources
	endpointsLister corev1listers.EndpointsLister
	revisionLister  listers.RevisionLister
	subsetEps       map[types.NamespacedName]*corev1.Endpoints
	revIDSet        RevIDSet
}


// Check that our Reconciler implements metricreconciler.Interface
var _ aepreconciler.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(_ context.Context, ActivationEndpoint *v1alpha1.ActivationEndpoint) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	activatorEps, err := r.endpointsLister.Endpoints(system.Namespace()).Get(networking.ActivatorServiceName)
	if err != nil {
		return logger.Infof("failed to get activator service endpoints: %w", err)
	}
	if len(activatorEps.Subsets) == 0 {
		return logger.Infof("Activator endpoints has no subsets")
	}

	addrs := make(sets.String, len(eps.Subsets[0].Addresses))

    revID := {Namespace: ActivationEndpoint.ObjectMeta.namespaces,
        Name: ActivationEndpoint.ObjectMeta.OwnerReferences.name
	}
	
    ActivationEpNum := subsetActivationEndpointsNum(activatorEps)
    subEps := subsetActivationEndpoints(activatorEps, revID, ActivationEpNum)

    r.update(revID, ActivationEpNum, subEps)

	ActivationEndpoint.Status.MarkActivationEndpointReady()

	return nil
}

func (r *reconciler) subsetActivationEndpointsNum(eps *corev1.Endpoints) int {

    return min(len(eps.Subsets[0].Addresses, 3)
}

func (r *reconciler) subsetActivationEndpoints(eps *corev1.Endpoints, revID *type.NamespacedName, n int) *corev1.Endpoints {
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

	// Copy the informer's copy, so we can filter it out.
	neps := eps.DeepCopy()

	if subsets := r.subEps[revID] {
		if len(subsets) >= n {
			neps.Subsets = subsets[:n] 
		}
		else len(subsets) < n {
            for i := 0; i< (n-len(subsets)); {
				for _, ss := range eps.Subsets {
					if r.revIDSet[ss.Addresses.IP] {
						continue
					}
					else {
						neps.Subsets.append(ss)
						i++
					}
				}
			} 
		}
	}
	else {
		r, w := 0, 0
		for r < len(neps.Subsets) {
			ss := neps.Subsets[r]
			// And same algorithm internally.
			ra, wa := 0, 0
			for ra < len(ss.Addresses) {
				if ! r.revIDSet[ss.Addresses[ra].IP] {
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

		neps.Subsets = neps.Subsets[:w]
	}

	return neps
}

func (r *reconciler) update(revID type.NamespacedName, ActivationEpNum int, eps *corev1.Endpoints) error {
	activatorEP := ActivatorEP{}

	for i := 0; i < len(eps.Subsets[0].Addresses); i++  {
		r.revIDSet[eps.Subsets[0].Addresses[i].IP].append(revID)
	}

    r.subsetEps[revID] = eps
}
