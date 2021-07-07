/*
Copyright 2018 The Knative Authors

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

package resources

import (
	"context"
	"math"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/autoscaling"
	autoscalingv1alpha1 "knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
	"knative.dev/serving/pkg/reconciler/revision/config"
	"knative.dev/serving/pkg/reconciler/revision/resources/names"
	"knative.dev/serving/pkg/resources/revision"
)

// MakeAEP makes a ActivationEndpoint resource from a revision.
func MakeAEP(ctx context.Context, rev *v1.Revision, *config.Config) *autoscalingv1alpha1.ActivationEndpoint {
	return &autoscalingv1alpha1.ActivationEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.AEP(rev),
			Namespace:       rev.Namespace,
			Labels:          revision.MakeLabels(rev),
			Annotations:     revision.MakeAnnotations(rev),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Spec: autoscalingv1alpha1.ActivationEndpointSpec{
			desiredActivationEpNum: computeActivatorEpNum(ctx, rev)
		},
	}
}

func computeActivatorEpNum(ctx context.Context, rev *v1.Revision) int {
    cfgs := config.FromContext(ctx)
    annotations := rev.GetAnnotations()

    numAct := cfg.Autoscaler.MaxNodeSelection
    if v, ok := annotations[autoscaling.MaxNodeSelectionKey]; ok {
        numAct = v
    }

    tbc := cfg.Autoscaler.TargetBurstCapacity
	if v, ok := annotations[autoscaling.TargetBurstCapacityKey]; ok {
		tbc = v
	}

    if tbc != -1 {
		cc := rev.spec.GetContainerConcurrency()
        numAct = int32(math.Max(1, (tbc - cc)))
    }

    return numAct
}