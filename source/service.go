package source

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	serviceNameAnnotationKey    = "grpc-http-proxy.alpha.mercari.com/grpc-service"
	serviceVersionAnnotationKey = "grpc-http-proxy.alpha.mercari.com/grpc-service-version"
)

// Service watches the Kubernetes API and updates records when there are changes to Service resources
type Service struct {
	*Records
	logger    *zap.Logger
	informer  cache.SharedIndexInformer
	namespace string
	lister    corelisters.ServiceLister
	queue     workqueue.RateLimitingInterface
}

// NewService creates a new Service source
func NewService(
	client clientset.Interface,
	namespace string,
	l *zap.Logger) *Service {

	opts := make([]informers.SharedInformerOption, 0)
	if namespace != "" {
		opts = append(opts, informers.WithNamespace(namespace))
	}
	infFactory := informers.NewSharedInformerFactoryWithOptions(client,
		30*time.Second, opts...)

	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second)
	k := &Service{
		Records:   NewRecords(),
		logger:    l,
		queue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, "Services"),
		namespace: namespace,
	}
	serviceInformer := infFactory.Core().V1().Services()
	k.informer = serviceInformer.Informer()
	k.lister = serviceInformer.Lister()
	eventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc, ok := obj.(*core.Service)
			if !ok {
				k.logger.Error(fmt.Sprintf("event for invalid object; got %T want *core.Service", obj))
				return
			}
			event := &Event{
				EventType: createEvent,
				Svc:       svc,
			}
			k.queue.AddRateLimited(event)
			k.logger.Debug(
				"add queued",
				zap.String("service", svc.Name),
				zap.Int("retries", k.queue.NumRequeues(event)),
				zap.Int("queue_length", k.queue.Len()),
			)
			return
		},
		DeleteFunc: func(obj interface{}) {
			svc, ok := obj.(*core.Service)
			if !ok {
				k.logger.Error(fmt.Sprintf("event for invalid object; got %T want *core.Service", obj))
				return
			}
			event := &Event{
				EventType: deleteEvent,
				Svc:       svc,
			}
			k.queue.AddRateLimited(event)
			k.logger.Debug(
				"delete queued",
				zap.String("service", svc.Name),
				zap.Int("retries", k.queue.NumRequeues(event)),
				zap.Int("queue_length", k.queue.Len()),
			)
			return
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSvc, ok := oldObj.(*core.Service)
			if !ok {
				k.logger.Error(fmt.Sprintf("event for invalid object; got %T want *core.Service", newObj))
				return
			}
			newSvc, ok := newObj.(*core.Service)
			if !ok {
				k.logger.Error(fmt.Sprintf("event for invalid object; got %T want *core.Service", newObj))
				return
			}
			k.logger.Debug(
				"adding service to queue",
				zap.String("service", newSvc.ObjectMeta.Name),
			)
			event := &Event{
				EventType: updateEvent,
				Svc:       newSvc,
				OldSvc:    oldSvc,
			}
			k.queue.AddRateLimited(event)
			k.logger.Debug(
				"update queued",
				zap.String("service", newSvc.Name),
				zap.Int("retries", k.queue.NumRequeues(event)),
				zap.Int("queue_length", k.queue.Len()),
			)
			return
		},
	}
	k.informer.AddEventHandler(eventHandler)

	return k
}

// Resolve resolves the FQDN for a backend providing the gRPC service specified
func (k *Service) Resolve(svc, version string) (*url.URL, error) {
	r, err := k.Records.GetRecord(svc, version)
	if err != nil {
		k.logger.Error("failed to resolve service",
			zap.String("service", svc),
			zap.String("version", version),
			zap.String("err", err.Error()))
		return nil, err
	}
	return r, nil
}

// Run starts the Service controller
func (k *Service) Run(stopCh <-chan struct{}) {
	go k.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, k.informer.HasSynced) {
		k.logger.Error("timed out waiting for caches to sync")
	}
	go wait.Until(k.runWorker, time.Second, stopCh)
}

func (k *Service) runWorker() {
	for k.processNextItem() {
	}
}

func (k *Service) processNextItem() bool {
	obj, quit := k.queue.Get()
	if quit {
		return false
	}
	err := func(obj interface{}) error {
		defer k.queue.Done(obj)
		evt, ok := obj.(*Event)
		if !ok {
			k.queue.Forget(obj)
			return errors.Errorf("expected Event in workqueue but got %#v", obj)
		}
		k.eventHandler(evt)
		return nil
	}(obj)
	if err != nil {
		k.logger.Error("failure in processing item",
			zap.String("err", err.Error()))
		return true
	}
	return true
}

func (k *Service) eventHandler(evt *Event) {
	switch evt.EventType {
	case createEvent:
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Debug("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}
		u, ok := k.constructURL(evt.Svc)
		if !ok {
			return
		}
		gRPCServiceNames := strings.Split(evt.Svc.Annotations[serviceNameAnnotationKey], ",")

		for _, svcName := range gRPCServiceNames {
			if metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceVersionAnnotationKey) {
				version := evt.Svc.Annotations[serviceVersionAnnotationKey]
				k.Records.SetRecord(svcName, version, u)
				continue
			}
			k.Records.SetRecord(svcName, "", u)
		}
	case deleteEvent:
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Debug("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}
		u, ok := k.constructURL(evt.Svc)
		if !ok {
			return
		}
		gRPCServiceNames := strings.Split(evt.Svc.Annotations[serviceNameAnnotationKey], ",")

		for _, svcName := range gRPCServiceNames {
			version := evt.Svc.Annotations[serviceVersionAnnotationKey]
			k.Records.RemoveRecord(svcName, version, u)
		}
	case updateEvent:
		// Service versions before and after update do not have annotations
		// Skip service and return
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) &&
			!metav1.HasAnnotation(evt.OldSvc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Debug("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}

		// Service versions before and after update both have gRPC service annotations
		if metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) &&
			metav1.HasAnnotation(evt.OldSvc.ObjectMeta, serviceNameAnnotationKey) {
			serviceNameAnnotationValue := evt.Svc.Annotations[serviceNameAnnotationKey]
			oldServiceNameAnnotationValue := evt.OldSvc.Annotations[serviceNameAnnotationKey]
			version := evt.Svc.Annotations[serviceVersionAnnotationKey]
			oldVersion := evt.OldSvc.Annotations[serviceVersionAnnotationKey]
			gRPCServiceNames := strings.Split(serviceNameAnnotationValue, ",")
			oldGRPCServiceNames := strings.Split(oldServiceNameAnnotationValue, ",")

			if oldServiceNameAnnotationValue != serviceNameAnnotationValue {
				// gRPC service name was changed
				oldURL, ok := k.constructURL(evt.OldSvc)
				if !ok {
					return
				}
				for _, oldSvcName := range oldGRPCServiceNames {
					k.Records.RemoveRecord(oldSvcName, oldVersion, oldURL)
				}
				u, ok := k.constructURL(evt.Svc)
				if !ok {
					return
				}
				for _, svcName := range gRPCServiceNames {
					k.Records.SetRecord(svcName, version, u)
				}
				return
			}

			if version != oldVersion {
				// version annotation was changed
				oldURL, ok := k.constructURL(evt.OldSvc)
				if !ok {
					return
				}
				for _, oldSvcName := range oldGRPCServiceNames {
					k.Records.RemoveRecord(oldSvcName, oldVersion, oldURL)
				}
				u, ok := k.constructURL(evt.Svc)
				if !ok {
					return
				}
				for _, svcName := range gRPCServiceNames {
					k.Records.SetRecord(svcName, version, u)
				}
				return
			}

			if k.areServicesMissing(gRPCServiceNames, version) {
				// Some records is missing, so reprocess all services in annotation
				for _, svcName := range gRPCServiceNames {
					u, ok := k.constructURL(evt.Svc)
					if !ok {
						return
					}
					k.Records.SetRecord(svcName, version, u)
				}
				return
			}

			if !reflect.DeepEqual(evt.Svc.Spec.Ports, evt.OldSvc.Spec.Ports) {
				// ports were updated
				oldURL, ok := k.constructURL(evt.OldSvc)
				if !ok {
					return
				}
				for _, oldSvcName := range oldGRPCServiceNames {
					k.Records.RemoveRecord(oldSvcName, oldVersion, oldURL)
				}
				u, ok := k.constructURL(evt.Svc)
				if !ok {
					return
				}
				for _, svcName := range gRPCServiceNames {
					k.Records.SetRecord(svcName, version, u)
				}
				return
			}

			// do nothing, since no annotations were changed
			return
		}

		// gRPC service annotation was removed from the Service
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			oldURL, ok := k.constructURL(evt.OldSvc)
			if !ok {
				return
			}
			oldGRPCServiceNames := strings.Split(evt.OldSvc.Annotations[serviceNameAnnotationKey], ",")
			oldVersion := evt.OldSvc.Annotations[serviceVersionAnnotationKey]
			for _, oldSvcName := range oldGRPCServiceNames {
				k.Records.RemoveRecord(oldSvcName, oldVersion, oldURL)
			}
			return
		}

		// gRPC service annotation was added to the Service
		gRPCServiceNames := strings.Split(evt.Svc.Annotations[serviceNameAnnotationKey], ",")
		version := evt.Svc.Annotations[serviceVersionAnnotationKey]
		u, ok := k.constructURL(evt.Svc)
		if !ok {
			return
		}
		for _, svcName := range gRPCServiceNames {
			k.Records.SetRecord(svcName, version, u)
		}
	}
}

// constructURL is a helper method that constructs URLs by obtaining necessary information from the Service
func (k *Service) constructURL(svc *core.Service) (*url.URL, bool) {
	port, ok := selectPort(svc.Spec.Ports)
	if !ok {
		k.logger.Debug("not adding new version of service because of invalid ports",
			zap.String("namespace", svc.Namespace),
			zap.String("name", svc.Name),
		)
		return nil, false
	}
	rawurl := fmt.Sprintf("%s.%s.svc.cluster.local:%d",
		svc.Name,
		svc.Namespace,
		port,
	)
	u, err := url.Parse(rawurl)
	if err != nil {
		k.logger.Error("failure in processing change to Service",
			zap.String("namespace", svc.Namespace),
			zap.String("name", svc.Name),
			zap.String("err", err.Error()),
		)
		return nil, false
	}
	return u, true
}

// selectPort selects a port from the Service
// * if there are zero ports, the second return value will be false
// * if there are exactly one port, that will be returned
// * if there are more than one port, the first one whose name has the
//   prefix "grpc" will be returned
// * if there are no ports with the "grpc" prefix, the second return value will be false
func selectPort(ports []core.ServicePort) (int32, bool) {
	if len(ports) == 0 {
		return 0, false
	}
	if len(ports) == 1 {
		return ports[0].Port, true
	}
	for _, p := range ports {
		if strings.HasPrefix(p.Name, "grpc") {
			return p.Port, true
		}
	}
	return 0, false
}

// areServicesMissing is a helper method that check if any services are missing from the records
func (k *Service) areServicesMissing(serviceNames []string, version string) bool {
	for _, svcName := range serviceNames {
		if !k.Records.RecordExists(svcName, version) {
			return true
		}
	}
	return false
}

// Event is an change event to a Kubernetes Service
type Event struct {
	EventType
	Svc    *core.Service
	OldSvc *core.Service
}

// EventType is the type of an event
type EventType string

const (
	createEvent EventType = "CREATE"
	updateEvent EventType = "UPDATE"
	deleteEvent EventType = "DELETE"
)
