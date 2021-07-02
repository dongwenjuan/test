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

package localscheduler

import (
    "context"
    "time"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
	revisioninformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/revision"
	aepinformer "knative.dev/serving/pkg/client/injection/informers/autoscaling/v1alpha1/activationendpoint"

	v1 "knative.dev/serving/pkg/apis/serving/v1"
	lsresources "knative.dev/serving/pkg/activator/localscheduler/resources"
	servinglisters "knative.dev/serving/pkg/client/listers/serving/v1"
	aslisters "knative.dev/serving/pkg/client/listers/autoscaling/v1alpha1"
	"knative.dev/serving/pkg/reconciler/revision/config"
	"knative.dev/serving/pkg/reconciler/revision/resources/names"
	"knative.dev/serving/pkg/resources"
	"knative.dev/serving/pkg/resources/revision"

)

const cleanupInterval = time.Second

type LocalScheduler struct {
    ctx              context.Context
	kubeclient       kubernetes.Interface
	revIDCh          chan types.NamespacedName
	revisionLister   servinglisters.RevisionLister
	aepLister        aslisters.ActivationEndpointLister
	nodeName         string
	podIP            string
	cfgs             *config.Config
	logger           *zap.SugaredLogger
	localPod         map[types.NamespacedName]string
}

func NewLocalScheduler(ctx context.Context, nodename, podIP string, revIDCh chan types.NamespacedName) *LocalScheduler {
    return &LocalScheduler{
        ctx:               ctx,
        kubeclient:        kubeclient.Get(ctx),
        revisionLister:    revisioninformer.Get(ctx).Lister(),
        aepLister:         aepinformer.Get(ctx).Lister(),
        revIDCh:           revIDCh,
        nodeName:          nodename,
        podIP:             podIP,
        cfgs:              config.FromContext(ctx),
        logger:            logging.FromContext(ctx),
        localPod:          make(map[types.NamespacedName]string),
    }
}

func (ls *LocalScheduler) Run(stopCh <-chan struct{}) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
    ls.run(stopCh, ticker.C)
}

func (ls *LocalScheduler) run(stopCh <-chan struct{}, cleanupCh <-chan time.Time) {
    for {
        select {
        case revID := <-ls.revIDCh:
            // Scaled up.
            go ls.nodeLocalScheduler(revID)
        case now := <-cleanupCh:
            go ls.timerCleanupLocalPod()
        case <-stopCh:
            go ls.stopLocalPod()
            return
        }
    }
}

func (ls *LocalScheduler) nodeLocalScheduler(revID types.NamespacedName) {
	sks, err := ls.kubeclient.v1alpha1().ServerlessService(revID.Namespace).Get(revID.Name)
	if err != nil {
        ls.logger.Fatal("Error get SKS : ", err)
        return
    }
    if sks.IsReady() {
        ls.logger.Info("SKS is ready, do not need to start local pod!")
        return
    }

    rev, err := ls.revisionLister.Revisions(revID.Namespace).Get(revID.Name)
    if err != nil {
        ls.logger.Fatal("Error get revision : ", err)
        return
    }
    pod, err := ls.createPodObject(rev, ls.cfgs)
	if err != nil {
	    ls.logger.Fatal("Error createPodObject : ", err)
		return
	}

    p, err := ls.kubeclient.CoreV1().Pod(pod.Namespace).Create(ls.ctx, pod, metav1.CreateOptions{})
	if err != nil {
        ls.logger.Fatal("Error apply PodObject : ", err)
		return
	}

	ls.localPod[revID] = p.Name
}

func (ls *LocalScheduler) createPodObject(rev *v1.Revision, cfg *config.Config) (*corev1.Pod, error) {
	podSpec, err := revision.MakePodSpec(rev, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create PodSpec: %w", err)
	}
	podSpec.NodeName = ls.nodeName
	PodName := lsresources.Pod(rev)

	labels := revision.MakeLabels(rev)
	anns := revision.MakeAnnotations(rev)

    return &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:        PodName,
            Namespace:   rev.Namespace,
            Labels:      labels,
            Annotations: anns,
        },
        Spec: *podSpec,
    }, nil
}

func (ls *LocalScheduler) timerCleanupLocalPod() {
    needToDelete := make([]types.NamespacedName, 0)

    for k, v := range ls.localPod {
        rev, err := ls.revisionLister.Revisions(k.Namespace).Get(k.Name)
        if err != nil {
            ls.logger.Fatal("Error get revision : ", err)
            return
        }

        if ls.isNeedToDelete(rev) {
            ls.cleanupLocalPod(k.Namespace, v)
            needToDelete = append(needToDelete, k)
        }
    }

    for _, v := range needToDelete {
        delete(ls.localPod, v)
    }
}

func (ls *LocalScheduler) stopLocalPod() {
    for k, v := range ls.localPod {
        ls.cleanupLocalPod(k.Namespace, v)
    }
}

func (ls *LocalScheduler) cleanupLocalPod(namespace, podName string) error {
    err := ls.kubeclient.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
    if err != nil {
        ls.logger.Fatal("Delete Local Pod Error : ", err)
    }
    return nil
}

func (ls *LocalScheduler) isNeedToDelete(rev *v1.Revision) bool {
    aepName := names.AEP(rev)
	if aep, err := ls.kubeclient.AutoscalingV1alpha1().ActivationEndpoint(rev.Namespace).Get(ls.ctx, aepName, metav1.UpdateOptions{}); err != nil {
        ls.logger.Fatal("Get ActivationEndpoint ERROR : ", err)
	}
    if ! resources.Include(aep.Status.subsets, ls.podIP) {
        return true
    }

    pa, err := ls.kubeclient.v1alpha1().PodAutoscaler(rev.Namespace).Get(ls.ctx, rev.Name, metav1.CreateOptions{
        LabelSelector: revision.MakeSelector(rev)})
    if err != nil {
        ls.logger.Fatal("Get PodAutoscaler ERROR : ", err)
        return false
    }
    asNum := pa.Status.ActualScale
    dsNum := pa.Status.DesiredScale
    if 0 < asNum - dsNum < ActivationNum {
        return true
    }

    return false
}