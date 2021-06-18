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

package activator

import (
	"context"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"knative.dev/serving/pkg/autoscaler/metrics"
)

// RawSender sends raw byte array messages with a message type
// (implemented by gorilla/websocket.Socket).
type LocalScheduler interface {

}



type LocalScheduler struct {
	kubeclient    kubernetes.Interface
	client        clientset.Interface
}

func NewLocalScheduler(logger *zap.SugaredLogger, lsCh <-chan asmetrics.StatMessage) {

}

func (ls *LocalScheduler) CreatePod(logger *zap.SugaredLogger, lsCh <-chan struct{}) {

	podSpec, err := makePodSpec(rev, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create PodSpec: %w", err)
	}
	labels := makeLabels(rev)
	anns := makeAnnotations(rev)

	Template: corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
			Annotations: anns,
		},
		Spec: *podSpec,
	},

}

func (ls *LocalScheduler) DeletePod(logger *zap.SugaredLogger, lsCh <-chan struct{}) {


}