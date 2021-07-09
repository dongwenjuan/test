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
	"strconv"

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
func MakeAEP(ctx context.Context, rev *v1.Revision) *autoscalingv1alpha1.ActivationEndpoint {
	return &autoscalingv1alpha1.ActivationEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.AEP(rev),
			Namespace:       rev.Namespace,
			Labels:          revision.MakeLabels(rev),
			Annotations:     revision.MakeAnnotations(rev),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Spec: autoscalingv1alpha1.ActivationEndpointSpec{
			DesiredActivationEpNum: computeActivatorEpNum(ctx, rev),
		},
	}
}

func computeActivatorEpNum(ctx context.Context, rev *v1.Revision) int32 {
    cfgs := config.FromContext(ctx)
    annotations := rev.GetAnnotations()

    numAct := cfgs.Autoscaler.MaxNodeSelection
    if v, ok := annotations[autoscaling.MaxNodeSelectionKey]; ok {
    	if iv, err := strconv.ParseInt(v, 10, 32); err == nil {
    	    numAct = int32(iv)
    	}
    }

    tbc := cfgs.Autoscaler.TargetBurstCapacity
	if v, ok := annotations[autoscaling.TargetBurstCapacityKey]; ok {
		if fv, err := strconv.ParseFloat(v, 64); err != nil || fv < 0 && fv != -1 {
		} else {
		    tbc = fv
		}
	}

    if tbc != -1 {
		cc := rev.Spec.GetContainerConcurrency()
        numAct = int32(math.Max(1, (math.Round(tbc) - float64(cc))))
    }

    return numAct
}