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

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/autoscaler/metrics"

)


type LocalScheduler struct {
	kubeclient       kubernetes.Interface
	revIDCh          chan *types.NamespacedName
	revisionLister   servinglisters.RevisionLister
	nodeName         string
	cfgs             *config.Config
	logger           *zap.SugaredLogger
	isLocalPod       map[types.NamespacedName]bool
}

func NewLocalScheduler(ctx context.Context, nodename string, revIDCh <-chan *types.NamespacedName) *LocalScheduler {
    return &LocalScheduler{
        kubeClient:        kubeclient.Get(ctx),
        revisionLister:    revisionInformer.Lister(),
        revIDCh:           revIDCh,
        nodeName:          nodename,
        cfgs:              config.FromContext(ctx)
        logger:            logging.FromContext(ctx),
        isLocalPod:        make(map[types.NamespacedName]bool),
    }
}

func (ls *LocalScheduler) Run(stopCh <-chan struct{}) {
    for {
        select {
        case revID := <-ls.revIDCh:
            // Scaled up.
            go ls.RunNodeLocalScheduler(revID)
        case <-stopCh:

            return
        }
    }
}

func (ls *LocalScheduler) RunNodeLocalScheduler(ctx context.Context, revID *types.NamespacedName) {

    rev, err := ls.revisionLister.Revisions(revID.Namespace).Get(revID.Name)
    if err != nil {
        return nil, err
    }

    pod, err := ls.CreatePodTemplate(rev, ls.cfgs)
	if err != nil {
		return nil, err
	}

    p, err := ls.kubeclient.CoreV1().Pod(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	ls.isLocalPod[revID] = true
}

func (ls *LocalScheduler) CreatePodTemplate(rev *v1.Revision, cfg *config.Config) (*corev1.Pod, error) {

	podSpec, err := makePodSpec(rev, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create PodSpec: %w", err)
	}
	labels := makeLabels(rev)
	anns := makeAnnotations(rev)

    return &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name: ,
            Namespace:   rev.Namespace,
            Labels:      labels,
            Annotations: anns,
        },
        Spec: *podSpec,
    }, nil
}

func (ls *LocalScheduler) DeletePod(rev *v1.Revision) {

    for k, v := range ls.isLocalPod {
        if v {
            err := client.CoreV1().Pods(k.Namespace).Delete(context.TODO(), k.Name, metav1.DeleteOptions{
                FieldSelector: "spec.nodeName=" + ls.nodeName,
                LabelSelector: makeSelector(rev),})
        }
    }
}