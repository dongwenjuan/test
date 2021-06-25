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
    "time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/autoscaler/metrics"

)


const cleanupInterval = time.Second

const (

    PodNotStart    = "NotStart"
    PodStarted     = "Started"
	PodInitialized = "Initialized"
	PodReady       = "Ready"
)

type LocalScheduler struct {
	kubeclient       kubernetes.Interface
	revIDCh          chan *types.NamespacedName
	revisionLister   servinglisters.RevisionLister
	nodeName         string
	cfgs             *config.Config
	logger           *zap.SugaredLogger
	LocalPodStatus   map[types.NamespacedName]string
}

func NewLocalScheduler(ctx context.Context, nodename string, revIDCh <-chan *types.NamespacedName) *LocalScheduler {
    return &LocalScheduler{
        kubeClient:        kubeclient.Get(ctx),
        revisionLister:    revisionInformer.Lister(),
        revIDCh:           revIDCh,
        nodeName:          nodename,
        cfgs:              config.FromContext(ctx)
        logger:            logging.FromContext(ctx),
        LocalPodStatus:    make(map[types.NamespacedName]string),
    }
}

func (ls *LocalScheduler) Run(stopCh <-chan struct{}) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
    ls.run(stopCh, ticker.C)
}

func (ls *LocalScheduler) run(stopCh <-chan struct{}, reportCh <-chan time.Time) {
    for {
        select {
        case revID := <-ls.revIDCh:
            // Scaled up.
            go ls.RunNodeLocalScheduler(revID)
        case now := <-ticker.C
            go ls.TimerCleanup()
        case <-stopCh:
            go ls.DeleteNodeLocalSchedulerPods()
            return
        }
    }
}

func (ls *LocalScheduler) RunNodeLocalScheduler(ctx context.Context, revID *types.NamespacedName) {
	sks, err := c.SKSLister.ServerlessServices(revID.Namespace).Get(revID.Name)
	if err != nil {
        ls.logger.Fatal("Error get SKS : ", err)
        return
    }
    if sks.IsReady() {
        ls.logger.info("SKS is ready, do not need to start local pod!")
        return
    }

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

	ls.LocalPodStatus[revID] = PodStarted
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

func (ls *LocalScheduler) TimerCleanup() {
    for k, v := range ls.LocalPodStatus {
        if v == PodStart {
            pa := ls.kubeclient.v1alpha1().pa(k.Namespace).Get(ctx, k.Nme, metav1.CreateOptions{})
            asNum := pa.Status.ActualScale
            dsNum := pa.Status.DesiredScale
            if 0 < asNum - dsNum < ActivationNum {
                ls.deletePod(k)
            }
        }
    }
}

func (ls *LocalScheduler) deletePod(revID *type.NamespacedName) error {

    err := ls.kubeclient.CoreV1().Pods(revID.Namespace).Delete(context.TODO(), revID.Name, metav1.DeleteOptions{
        FieldSelector: "spec.nodeName=" + ls.nodeName,
        LabelSelector: makeSelector(rev),})

    return nil
}

func (ls *LocalScheduler) DeleteNodeLocalSchedulerPods() {
    for k, v := range ls.LocalPodStatus {
        if v != PodNotStart {
            ls.deletePod(k)
        }
    }
}

