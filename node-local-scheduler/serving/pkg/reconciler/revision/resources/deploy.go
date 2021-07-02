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
	"fmt"
	"strconv"

	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
	"knative.dev/serving/pkg/apis/autoscaling"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
	"knative.dev/serving/pkg/reconciler/revision/config"
	"knative.dev/serving/pkg/reconciler/revision/resources/names"
	"knative.dev/serving/pkg/resources/revision"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MakeDeployment constructs a K8s Deployment resource from a revision.
func MakeDeployment(rev *v1.Revision, cfg *config.Config) (*appsv1.Deployment, error) {
	podSpec, err := revision.MakePodSpec(rev, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create PodSpec: %w", err)
	}

	replicaCount := cfg.Autoscaler.InitialScale
	ann, found := rev.Annotations[autoscaling.InitialScaleAnnotationKey]
	if found {
		// Ignore errors and no error checking because already validated in webhook.
		rc, _ := strconv.ParseInt(ann, 10, 32)
		replicaCount = int32(rc)
	}

	labels := revision.MakeLabels(rev)
	anns := revision.MakeAnnotations(rev)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.Deployment(rev),
			Namespace:       rev.Namespace,
			Labels:          labels,
			Annotations:     anns,
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:                ptr.Int32(replicaCount),
			Selector:                revision.MakeSelector(rev),
			ProgressDeadlineSeconds: ptr.Int32(int32(cfg.Deployment.ProgressDeadline.Seconds())),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: anns,
				},
				Spec: *podSpec,
			},
		},
	}, nil
}
