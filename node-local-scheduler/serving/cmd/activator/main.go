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

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	// Injection related imports.
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/serving/pkg/activator"
	revisioninformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/revision"

	"k8s.io/apimachinery/pkg/util/wait"

	network "knative.dev/networking/pkg"
	"knative.dev/pkg/configmap"
	configmapinformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	pkglogging "knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/pkg/metrics"
	pkgnet "knative.dev/pkg/network"
	"knative.dev/pkg/profiling"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
	"knative.dev/pkg/tracing"
	tracingconfig "knative.dev/pkg/tracing/config"
	"knative.dev/pkg/version"
	activatorconfig "knative.dev/serving/pkg/activator/config"
	activatorhandler "knative.dev/serving/pkg/activator/handler"
	activatorls "knative.dev/serving/pkg/activator/localscheduler"
	activatornet "knative.dev/serving/pkg/activator/net"
	asmetrics "knative.dev/serving/pkg/autoscaler/metrics"
    grpcclient "knative.dev/serving/pkg/grpc/client"
	pkghttp "knative.dev/serving/pkg/http"
	"knative.dev/serving/pkg/logging"
	"knative.dev/serving/pkg/networking"
)

const (
	component = "activator"

	// The port on which autoscaler WebSocket server listens.
	autoscalerPort = ":8080"
)

type config struct {
	PodName   string `split_words:"true" required:"true"`
	PodIP     string `split_words:"true" required:"true"`
	NodeName  string `split_words:"true" required:"true"`

	// These are here to allow configuring higher values of keep-alive for larger environments.
	// TODO: run loadtests using these flags to determine optimal default values.
	MaxIdleProxyConns        int `split_words:"true" default:"1000"`
	MaxIdleProxyConnsPerHost int `split_words:"true" default:"100"`
}

func main() {
	// Set up a context that we can cancel to tell informers and other subprocesses to stop.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Report stats on Go memory usage every 30 seconds.
	metrics.MemStatsOrDie(ctx)

	cfg := injection.ParseAndGetRESTConfigOrDie()

	log.Printf("Registering %d clients", len(injection.Default.GetClients()))
	log.Printf("Registering %d informer factories", len(injection.Default.GetInformerFactories()))
	log.Printf("Registering %d informers", len(injection.Default.GetInformers()))

	ctx, informers := injection.Default.SetupInformers(ctx, cfg)

	var env config
	if err := envconfig.Process("", &env); err != nil {
		log.Fatal("Failed to process env: ", err)
	}

	kubeClient := kubeclient.Get(ctx)

	// We sometimes startup faster than we can reach kube-api. Poll on failure to prevent us terminating
	var err error
	if perr := wait.PollImmediate(time.Second, 60*time.Second, func() (bool, error) {
		if err = version.CheckMinimumVersion(kubeClient.Discovery()); err != nil {
			log.Print("Failed to get k8s version ", err)
		}
		return err == nil, nil
	}); perr != nil {
		log.Fatal("Timed out attempting to get k8s version: ", err)
	}

	// Set up our logger.
	loggingConfig, err := sharedmain.GetLoggingConfig(ctx)
	if err != nil {
		log.Fatal("Error loading/parsing logging configuration: ", err)
	}

	logger, atomicLevel := pkglogging.NewLoggerFromConfig(loggingConfig, component)
	logger = logger.With(zap.String(logkey.ControllerType, component),
		zap.String(logkey.Pod, env.PodName))
	ctx = pkglogging.WithLogger(ctx, logger)
	defer flush(logger)

	// Run informers instead of starting them from the factory to prevent the sync hanging because of empty handler.
	if err := controller.StartInformers(ctx.Done(), informers...); err != nil {
		logger.Fatalw("Failed to start informers", zap.Error(err))
	}

	logger.Info("Starting the knative activator")

	// Create the transport used by both the activator->QP probe and the proxy.
	// It's important that the throttler and the activatorhandler share this
	// transport so that throttler probe connections can be reused after probing
	// (via keep-alive) to send real requests, avoiding needing an extra
	// reconnect for the first request after the probe succeeds.
	logger.Debugf("MaxIdleProxyConns: %d, MaxIdleProxyConnsPerHost: %d", env.MaxIdleProxyConns, env.MaxIdleProxyConnsPerHost)
	transport := pkgnet.NewProxyAutoTransport(env.MaxIdleProxyConns, env.MaxIdleProxyConnsPerHost)

	// Start throttler.
	throttler := activatornet.NewThrottler(ctx, env.PodIP)
	go throttler.Run(ctx, transport)

	oct := tracing.NewOpenCensusTracer(tracing.WithExporterFull(networking.ActivatorServiceName, env.PodIP, logger))

	tracerUpdater := configmap.TypeFilter(&tracingconfig.Config{})(func(name string, value interface{}) {
		cfg := value.(*tracingconfig.Config)
		if err := oct.ApplyConfig(cfg); err != nil {
			logger.Errorw("Unable to apply open census tracer config", zap.Error(err))
			return
		}
	})

	// Set up our config store
	configMapWatcher := configmapinformer.NewInformedWatcher(kubeClient, system.Namespace())
	configStore := activatorconfig.NewStore(logger, tracerUpdater)
	configStore.WatchConfigs(configMapWatcher)

	statCh := make(chan []asmetrics.StatMessage)
	defer close(statCh)

    // Open a gRPC connection to the autoscaler
	autoscalerEndpoint := fmt.Sprintf("%s.%s.svc.%s%s", "autoscaler", system.Namespace(), pkgnet.GetClusterDomainName(), autoscalerPort)
	grpcclient, err := grpcclient.NewClient(autoscalerEndpoint)
	if err != nil {
		logger.Errorw("could not connect: %s", zap.Error(err))
	}
	defer grpcclient.Close()

	statSink := grpcclient.StatMsg
	go activator.ReportStats(logger, statSink, statCh)

	lpActionCh := make(chan activatorls.LocalPodAction)
	defer close(lpActionCh)

    LocalScheduler := activatorls.NewLocalScheduler(ctx, env.NodeName, env.PodIP, lpActionCh, logger)
	go LocalScheduler.Run(ctx.Done())

	// Create and run our concurrency reporter
	concurrencyReporter := activatorhandler.NewConcurrencyReporter(ctx, env.PodName, statCh, lpActionCh)
	go concurrencyReporter.Run(ctx.Done())

	// Create activation handler chain
	// Note: innermost handlers are specified first, ie. the last handler in the chain will be executed first
	var ah http.Handler = activatorhandler.New(ctx, throttler, transport)
	ah = concurrencyReporter.Handler(ah)
	ah = tracing.HTTPSpanMiddleware(ah)
	ah = configStore.HTTPMiddleware(ah)
	reqLogHandler, err := pkghttp.NewRequestLogHandler(ah, logging.NewSyncFileWriter(os.Stdout), "",
		requestLogTemplateInputGetter(revisioninformer.Get(ctx).Lister()), false /*enableProbeRequestLog*/)
	if err != nil {
		logger.Fatalw("Unable to create request log handler", zap.Error(err))
	}
	ah = reqLogHandler

	// NOTE: MetricHandler is being used as the outermost handler of the meaty bits. We're not interested in measuring
	// the healthchecks or probes.
	ah = activatorhandler.NewMetricHandler(env.PodName, ah)
	ah = activatorhandler.NewContextHandler(ctx, ah)

	// Network probe handlers.
	ah = &activatorhandler.ProbeHandler{NextHandler: ah}
	ah = network.NewProbeHandler(ah)

	// Set up our health check based on the health of stat sink and environmental factors.
	sigCtx, sigCancel := context.WithCancel(context.Background())
	hc := newHealthCheck(sigCtx, logger, grpcclient.Conn)
	ah = &activatorhandler.HealthHandler{HealthCheck: hc, NextHandler: ah, Logger: logger}

	profilingHandler := profiling.NewHandler(logger, false)
	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher.Watch(pkglogging.ConfigMapName(), pkglogging.UpdateLevelFromConfigMap(logger, atomicLevel, component))

	// Watch the observability config map
	configMapWatcher.Watch(metrics.ConfigMapName(),
		metrics.ConfigMapWatcher(ctx, component, nil /* SecretFetcher */, logger),
		updateRequestLogFromConfigMap(logger, reqLogHandler),
		profilingHandler.UpdateFromConfigMap)

	if err = configMapWatcher.Start(ctx.Done()); err != nil {
		logger.Fatalw("Failed to start configuration manager", zap.Error(err))
	}

	servers := map[string]*http.Server{
		"http1":   pkgnet.NewServer(":"+strconv.Itoa(networking.BackendHTTPPort), ah),
		"h2c":     pkgnet.NewServer(":"+strconv.Itoa(networking.BackendHTTP2Port), ah),
		"profile": profiling.NewServer(profilingHandler),
	}

	errCh := make(chan error, len(servers))
	for name, server := range servers {
		go func(name string, s *http.Server) {
			// Don't forward ErrServerClosed as that indicates we're already shutting down.
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("%s server failed: %w", name, err)
			}
		}(name, server)
	}

	sigCh := signals.SetupSignalHandler()

	// Wait for the signal to drain.
	select {
	case <-sigCh:
		logger.Info("Received SIGTERM")
		// Send a signal to let readiness probes start failing.
		sigCancel()
	case err := <-errCh:
		logger.Errorw("Failed to run HTTP server", zap.Error(err))
	}

	// The drain has started (we are now failing readiness probes).  Let the effects of this
	// propagate so that new requests are no longer routed our way.
	logger.Infof("Sleeping %v to allow K8s propagation of non-ready state", pkgnet.DefaultDrainTimeout)
	time.Sleep(pkgnet.DefaultDrainTimeout)
	logger.Info("Done waiting, shutting down servers.")

	// Drain outstanding requests, and stop accepting new ones.
	for _, server := range servers {
		server.Shutdown(context.Background())
	}
	logger.Info("Servers shutdown.")
}

func newHealthCheck(sigCtx context.Context, logger *zap.SugaredLogger, healthSink *grpc.ClientConn) func() error {
	once := sync.Once{}
	return func() error {
		select {
		// When we get SIGTERM (sigCtx done), let readiness probes start failing.
		case <-sigCtx.Done():
			once.Do(func() {
				logger.Info("Signal context canceled")
			})
			return errors.New("received SIGTERM from kubelet")
		default:
			logger.Debug("No signal yet.")
			if healthSink.GetState() == connectivity.TransientFailure {
				return errors.New("connection TransientFailure")
			}
			return nil
		}
	}
}

func flush(logger *zap.SugaredLogger) {
	logger.Sync()
	os.Stdout.Sync()
	os.Stderr.Sync()
	metrics.FlushExporter()
}
