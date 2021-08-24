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

	"knative.dev/pkg/metrics"
	pkgtracing "knative.dev/pkg/tracing/config"
	"knative.dev/serving/pkg/deployment"

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

const (
    cleanupInterval = time.Minute
    PodCreate       = "create"
    PodDelete       = "delete"
)

type LocalPodAction struct {
	RevID        types.NamespacedName
	Action       string
}

type LocalScheduler struct {
    ctx              context.Context
    cfgs             *config.Config

	kubeclient       kubernetes.Interface
    revisionLister   servinglisters.RevisionLister
    SKSLister        nlisters.ServerlessServiceLister
    paLister         aslisters.PodAutoscalerLister
	aepLister        aslisters.ActivationEndpointLister

	nodeName         string
	podIP            string
	lpActionCh       chan LocalPodAction
	localPod         map[types.NamespacedName]string
	logger           *zap.SugaredLogger
}

func NewLocalScheduler(ctx context.Context, nodename, podIP string, lpActionCh chan LocalPodAction, logger *zap.SugaredLogger) *LocalScheduler {
    kubeClient := kubeclient.Get(ctx)
    localScheduler := LocalScheduler{
        ctx:               ctx,

        kubeclient:        kubeClient,
        revisionLister:    revisioninformer.Get(ctx).Lister(),
        SKSLister:         sksinformer.Get(ctx).Lister(),
        paLister:          painformer.Get(ctx).Lister(),
        aepLister:         aepinformer.Get(ctx).Lister(),

        nodeName:          nodename,
        podIP:             podIP,
        lpActionCh:        lpActionCh,
        localPod:          make(map[types.NamespacedName]string),
        logger:            logger,
    }

    configMapWatcher := configmapinformer.NewInformedWatcher(kubeClient, system.Namespace())
	configMapWatcher.Watch(deployment.ConfigName, localScheduler.UpdateCfgs)
	configMapWatcher.Watch(metrics.ConfigMapName(), localScheduler.UpdateCfgs)
    configMapWatcher.Watch(pkgtracing.ConfigName, localScheduler.UpdateCfgs)

    configStore := config.NewStore(logger)
    configStore.WatchConfigs(configMapWatcher)
    if err := configMapWatcher.Start(ctx.Done()); err != nil {
        logger.Fatalw("Failed to start configmap watcher", zap.Error(err))
    }

    localScheduler.cfgs = configStore.Load()
    return &localScheduler
}

func (ls *LocalScheduler) UpdateCfgs(config *corev1.ConfigMap) {
    name := config.ObjectMeta.Name

    if ls.cfgs == nil {
        ls.logger.Info("For the first time to add configmap, the cfgs is not loaded, return!")
        return
    }
    switch name {
    case deployment.ConfigName:
        if dep, err := deployment.NewConfigFromConfigMap(config); err == nil {
            ls.cfgs.Deployment = dep.DeepCopy()
        }
    case metrics.ConfigMapName():
        if obs, err := metrics.NewObservabilityConfigFromConfigMap(config); err == nil {
            ls.cfgs.Observability = obs.DeepCopy()
        }
    case pkgtracing.ConfigName:
        if tr, err := pkgtracing.NewTracingConfigFromConfigMap(config); err == nil {
            ls.cfgs.Tracing = tr.DeepCopy()
        }
    default:
        ls.logger.Info("Do nothing for the configmap: %s change!", name)
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
        case lpActionCh := <-ls.lpActionCh:
		    switch lpActionCh.Action {
		    case PodCreate:
		        go func () {
		            if _, ok := ls.localPod[lpActionCh.RevID]; !ok {
		                ls.nodeLocalScheduler(lpActionCh.RevID)
		            }
		        }()
			case PodDelete:
			    go func() {
	                cfgAS := ls.cfgs.Autoscaler
	                if cfgAS.EnableScaleToZero {
			            time.Sleep(cfgAS.ScaleToZeroGracePeriod)
			            ls.cleanupLocalPod(lpActionCh.RevID)
			            delete(ls.localPod, lpActionCh.RevID)
			        }
	            }()
		    default:
			    ls.logger.Info("Error action for scheduler local pod!")
	        }
        case <-cleanupCh:
            go ls.timerCleanupLocalPod()
        case <-stopCh:
            go ls.stopLocalPod()
            return
        }
    }
}

func (ls *LocalScheduler) nodeLocalScheduler(revID types.NamespacedName) {
    ls.logger.Info("Start to scheduler local pod!")

    rev, err := ls.revisionLister.Revisions(revID.Namespace).Get(revID.Name)
    if err != nil {
        ls.logger.Errorw("Error get revision : ", zap.Error(err))
        return
    }

    paName := revisionname.PA(rev)
    sksName := asname.SKS(paName)
	sks, err := ls.SKSLister.ServerlessServices(revID.Namespace).Get(sksName)
	if err != nil {
        ls.logger.Errorw("Error get SKS : ", zap.Error(err))
        return
    }
    if sks.IsReady() {
        ls.logger.Info("SKS is ready, do not need to start local pod!")
        return
    }

    pod, err := ls.createPodObject(rev, ls.cfgs)
	if err != nil {
	    ls.logger.Errorw("Error createPodObject : ", zap.Error(err))
		return
	}

    p, err := ls.kubeclient.CoreV1().Pods(pod.Namespace).Create(ls.ctx, pod, metav1.CreateOptions{})
	if err != nil {
        ls.logger.Errorw("Error apply PodObject : ", zap.Error(err))
		return
	}

	ls.localPod[revID] = p.Name
}

func (ls *LocalScheduler) createPodObject(rev *v1.Revision, cfg *config.Config) (*corev1.Pod, error) {
	podSpec, err := revision.MakePodSpec(rev, cfg)
	if err != nil {
        ls.logger.Errorw("failed to create PodSpec: %w", zap.Error(err))
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
    for revID, _ := range ls.localPod {
        rev, err := ls.revisionLister.Revisions(revID.Namespace).Get(revID.Name)
        if err != nil {
            ls.logger.Errorw("Error get revision in timer cleanup : ", zap.Error(err))
            return
        }

        if ok := ls.isNeedToDelete(rev); ok {
            ls.logger.Info("Timer Cleanup: need to delete local pod!", revID)
            if err := ls.cleanupLocalPod(revID); err != nil {
                ls.logger.Errorw("failed to timer cleanup local Pod: %w", zap.Error(err))
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
    ls.logger.Info("Receive sigal to stop local pod!")

    for revID, _ := range ls.localPod {
        if err := ls.cleanupLocalPod(revID); err != nil {
            ls.logger.Errorw("failed to stop local Pod: %w", zap.Error(err))
        }
    }

    for revID, _ := range ls.localPod {
        delete(ls.localPod, revID)
    }

}

func (ls *LocalScheduler) cleanupLocalPod(revID types.NamespacedName) error {
    if podName, ok := ls.localPod[revID]; ok {
        ls.logger.Info("cleanupLocalPod, revID: !", revID)
        err := ls.kubeclient.CoreV1().Pods(revID.Namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
        if err != nil {
            ls.logger.Errorw("Delete Local Pod Error : ", zap.Error(err))
            return err
        }
    }
    return nil
}

func (ls *LocalScheduler) isNeedToDelete(rev *v1.Revision) bool {
    aepName := revisionname.AEP(rev)
    aep, err := ls.aepLister.ActivationEndpoints(rev.Namespace).Get(aepName)
    if err != nil {
        ls.logger.Errorw("Get ActivationEndpoint ERROR : ", zap.Error(err))
    }

    // This activator ep is no longer in the subset, need to stop local pod!
    if ! resources.Include(aep.Status.SubsetEPs, ls.podIP) {
        ls.logger.Info("Activator podIp is no longer in ActivationEndpoints, need to stop local Pod!")
        return true
    }

    paName := revisionname.PA(rev)
    pa, err := ls.paLister.PodAutoscalers(rev.Namespace).Get(paName)
    if err != nil {
        ls.logger.Errorw("Get PodAutoscaler ERROR : ", zap.Error(err))
        return false
    }

    asNum := pa.Status.GetActualScale()
    dsNum := pa.Status.GetDesiredScale()
    localPodNum := asNum - dsNum

    // Already scale cluster scheduler pods
    if dsNum > 0 && asNum > dsNum && localPodNum <= aep.Status.ActualActivationEpNum {
        return true
    }

    return false
}
