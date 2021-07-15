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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

    endpointsinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/endpoints"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

    kubeclient "knative.dev/pkg/client/injection/kube/client"
	aepinformer "knative.dev/serving/pkg/client/injection/informers/autoscaling/v1alpha1/activationendpoint"
	aepreconciler "knative.dev/serving/pkg/client/injection/reconciler/autoscaling/v1alpha1/activationendpoint"

    "knative.dev/serving/pkg/reconciler/revision/config"
)

// NewController initializes the controller and is called by the generated code.
// Registers eventhandlers to enqueue events.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)
	aepinformer := aepinformer.Get(ctx)
	endpointsInformer := endpointsinformer.Get(ctx)

	logger.Info("Setting up ConfigMap receivers")
	configStore := config.NewStore(logger.Named("config-store"))
	configStore.WatchConfigs(cmw)

	c := &reconciler{
		kubeclient: kubeclient.Get(ctx),
		endpointsLister: endpointsInformer.Lister(),
		subsetEps:   make(map[types.NamespacedName]*corev1.Endpoints),
	}

	impl := aepreconciler.NewImpl(ctx, c, func(*controller.Impl) controller.Options {
		return controller.Options{
			ConfigStore: configStore,
		}
	})

	logger.Info("Setting up event handlers")

	// Watch all the aep objects.
	aepinformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
