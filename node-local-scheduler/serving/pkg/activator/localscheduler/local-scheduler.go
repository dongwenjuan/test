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
    configmapinformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/system"
    nlisters "knative.dev/networking/pkg/client/listers/networking/v1alpha1"
    sksinformer "knative.dev/networking/pkg/client/injection/informers/networking/v1alpha1/serverlessservice"

	revisioninformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/revision"
	aepinformer "knative.dev/serving/pkg/client/injection/informers/autoscaling/v1alpha1/activationendpoint"
    painformer "knative.dev/serving/pkg/client/injection/informers/autoscaling/v1alpha1/podautoscaler"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
	lsresources "knative.dev/serving/pkg/activator/localscheduler/resources"
	servinglisters "knative.dev/serving/pkg/client/listers/serving/v1"
	aslisters "knative.dev/serving/pkg/client/listers/autoscaling/v1alpha1"
    "knative.dev/serving/pkg/reconciler/revision/config"
    asname "knative.dev/serving/pkg/reconciler/autoscaling/resources/names"
	revisionname "knative.dev/serving/pkg/reconciler/revision/resources/names"
	"knative.dev/serving/pkg/resources"
	"knative.dev/serving/pkg/resources/revision"
)

const cleanupInterval = time.Minute

type LocalScheduler struct {
    ctx              context.Context
	kubeclient       kubernetes.Interface
    revisionLister   servinglisters.RevisionLister
    SKSLister        nlisters.ServerlessServiceLister
    paLister         aslisters.PodAutoscalerLister
	aepLister        aslisters.ActivationEndpointLister

	nodeName         string
	podIP            string
	logger           *zap.SugaredLogger
	revIDCh          chan types.NamespacedName
	localPod         map[types.NamespacedName]string
}

func NewLocalScheduler(ctx context.Context, nodename, podIP string, revIDCh chan types.NamespacedName, logger *zap.SugaredLogger) *LocalScheduler {
    kubeClient := kubeclient.Get(ctx)
    configMapWatcher := configmapinformer.NewInformedWatcher(kubeClient, system.Namespace())
    configStore := config.NewStore(logger)
    configStore.WatchConfigs(configMapWatcher)
    ctx = configStore.ToContext(ctx)

    return &LocalScheduler{
        ctx:               ctx,
        kubeclient:        kubeclient.Get(ctx),
        revisionLister:    revisioninformer.Get(ctx).Lister(),
        SKSLister:         sksinformer.Get(ctx).Lister(),
        paLister:          painformer.Get(ctx).Lister(),
        aepLister:         aepinformer.Get(ctx).Lister(),
        revIDCh:           revIDCh,
        nodeName:          nodename,
        podIP:             podIP,
        logger:            logger,
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
        case <-cleanupCh:
            go ls.timerCleanupLocalPod()
        case <-stopCh:
            go ls.stopLocalPod()
            return
        }
    }
}

func (ls *LocalScheduler) nodeLocalScheduler(revID types.NamespacedName) {
    rev, err := ls.revisionLister.Revisions(revID.Namespace).Get(revID.Name)
    if err != nil {
        ls.logger.Fatal("Error get revision : ", err)
        return
    }
 
    paName := revisionname.PA(rev)
    sksName := asname.SKS(paName)
	sks, err := ls.SKSLister.ServerlessServices(revID.Namespace).Get(sksName)
	if err != nil {
        ls.logger.Fatal("Error get SKS : ", err)
        return
    }
    if sks.IsReady() {
        ls.logger.Info("SKS is ready, do not need to start local pod!")
        return
    }

    cfgs := config.FromContext(ls.ctx)
    pod, err := ls.createPodObject(rev, cfgs)
	if err != nil {
	    ls.logger.Fatal("Error createPodObject : ", err)
		return
	}

    p, err := ls.kubeclient.CoreV1().Pods(pod.Namespace).Create(ls.ctx, pod, metav1.CreateOptions{})
	if err != nil {
        ls.logger.Fatal("Error apply PodObject : ", err)
		return
	}

	ls.localPod[revID] = p.Name
}

func (ls *LocalScheduler) createPodObject(rev *v1.Revision, cfg *config.Config) (*corev1.Pod, error) {
	podSpec, err := revision.MakePodSpec(rev, cfg)
	if err != nil {
        ls.logger.Fatal("failed to create PodSpec: %w", err)
		return nil, err
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

    for revID, podName := range ls.localPod {
        rev, err := ls.revisionLister.Revisions(revID.Namespace).Get(revID.Name)
        if err != nil {
            ls.logger.Fatal("Error get revision in timer cleanup : ", err)
            return
        }

        if ok := ls.isNeedToDelete(rev); ok {
            if err := ls.cleanupLocalPod(revID.Namespace, podName); err != nil {
                ls.logger.Fatal("failed to timer cleanup local Pod: %w", err)
                return
            }
            needToDelete = append(needToDelete, revID)
        }
    }

    for _, revID := range needToDelete {
        delete(ls.localPod, revID)
    }
}

func (ls *LocalScheduler) stopLocalPod() {
    for revID, podName := range ls.localPod {
        if err := ls.cleanupLocalPod(revID.Namespace, podName); err != nil {
            ls.logger.Fatal("failed to stop local Pod: %w", err)
        }
    }
}

func (ls *LocalScheduler) cleanupLocalPod(namespace, podName string) error {
    err := ls.kubeclient.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
    if err != nil {
        ls.logger.Fatal("Delete Local Pod Error : ", err)
        return err
    }
    return nil
}

func (ls *LocalScheduler) isNeedToDelete(rev *v1.Revision) bool {
    aepName := revisionname.AEP(rev)
    aep, err := ls.aepLister.ActivationEndpoints(rev.Namespace).Get(aepName)
    if err != nil {
        ls.logger.Fatal("Get ActivationEndpoint ERROR : ", err)
    }

    // This activator ep is no longer in the subset, need to stop local pod!
    if ! resources.Include(aep.Status.SubsetEPs, ls.podIP) {
        return true
    }

    paName := revisionname.PA(rev)
    pa, err := ls.paLister.PodAutoscalers(rev.Namespace).Get(paName)
    if err != nil {
        ls.logger.Fatal("Get PodAutoscaler ERROR : ", err)
        return false
    }

    localPodNum := *(pa.Status.ActualScale) - *(pa.Status.DesiredScale)
    if localPodNum > 0 && localPodNum <= aep.Status.ActualActivationEpNum {
        return true
    }

    return false
}
