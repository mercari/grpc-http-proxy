package source

import (
	"fmt"
	"net/url"
	"sync"
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
	mutex     *sync.Mutex
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

	k := &Service{
		Records:   NewRecords(),
		logger:    l,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
		namespace: namespace,
		mutex:     &sync.Mutex{},
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
			k.queue.AddRateLimited(Event{
				EventType: createEvent,
				Svc:       svc,
			})
			return
		},
		DeleteFunc: func(obj interface{}) {
			svc, ok := obj.(*core.Service)
			if !ok {
				k.logger.Error(fmt.Sprintf("event for invalid object; got %T want *core.Service", obj))
				return
			}
			k.queue.AddRateLimited(Event{
				EventType: deleteEvent,
				Svc:       svc,
			})
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
			k.queue.AddRateLimited(Event{
				EventType: updateEvent,
				Svc:       newSvc,
				OldSvc:    oldSvc,
			})
			return
		},
	}
	k.informer.AddEventHandler(eventHandler)

	return k
}

// Resolve resolves the FQDN for a backend providing the gRPC service specified
func (k *Service) Resolve(svc, version string) (*url.URL, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()
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
		evt, ok := obj.(Event)
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

func (k *Service) eventHandler(evt Event) {
	rawurl := fmt.Sprintf("%s.%s.svc.cluster.local", evt.Svc.Name, evt.Svc.Namespace)
	u, err := url.Parse(rawurl)
	if err != nil {
		k.logger.Error("failure in processing change to Service",
			zap.String("namespace", evt.Svc.Namespace),
			zap.String("name", evt.Svc.Name),
			zap.String("err", err.Error()),
		)
		return
	}
	k.mutex.Lock()
	defer k.mutex.Unlock()
	switch evt.EventType {
	case createEvent:
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Debug("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}
		gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]

		if metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceVersionAnnotationKey) {
			version := evt.Svc.Annotations[serviceVersionAnnotationKey]
			k.Records.SetRecord(gRPCServiceName, version, u)
			return
		}
		k.Records.SetRecord(gRPCServiceName, "", u)
	case deleteEvent:
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Debug("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}
		gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]

		version := evt.Svc.Annotations[serviceVersionAnnotationKey]
		k.Records.RemoveRecord(gRPCServiceName, version, u)
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
			gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]
			oldGRPCServiceName := evt.OldSvc.Annotations[serviceNameAnnotationKey]
			version := evt.Svc.Annotations[serviceVersionAnnotationKey]
			oldVersion := evt.OldSvc.Annotations[serviceVersionAnnotationKey]

			if oldGRPCServiceName != gRPCServiceName {
				// gRPC service name was changed
				oldRawurl := fmt.Sprintf("%s.%s.svc.cluster.local",
					evt.OldSvc.Name,
					evt.OldSvc.Namespace,
				)
				oldURL, err := url.Parse(oldRawurl)
				if err != nil {
					k.logger.Error("failure in processing change to Service",
						zap.String("namespace", evt.Svc.Namespace),
						zap.String("name", evt.Svc.Name),
						zap.String("err", err.Error()),
					)
					return
				}
				k.Records.RemoveRecord(oldGRPCServiceName, oldVersion, oldURL)
				k.Records.SetRecord(gRPCServiceName, version, oldURL)
				return
			}

			if version != oldVersion {
				// version annotation was changed
				oldRawurl := fmt.Sprintf("%s.%s.svc.cluster.local",
					evt.OldSvc.Name,
					evt.OldSvc.Namespace,
				)
				oldURL, err := url.Parse(oldRawurl)
				if err != nil {
					k.logger.Error("failure in processing change to Service",
						zap.String("namespace", evt.Svc.Namespace),
						zap.String("name", evt.Svc.Name),
						zap.String("err", err.Error()),
					)
					return
				}
				k.Records.RemoveRecord(oldGRPCServiceName, oldVersion, oldURL)
				k.Records.SetRecord(gRPCServiceName, version, u)
				return
			}

			if !k.Records.RecordExists(gRPCServiceName, version) {
				// Record is missing, so add it
				k.Records.SetRecord(gRPCServiceName, version, u)
				return
			}

			// do nothing, since no annotations were changed
			return
		}

		// gRPC service annotation was removed from the Service
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			oldRawurl := fmt.Sprintf("%s.%s.svc.cluster.local",
				evt.OldSvc.Name,
				evt.OldSvc.Namespace,
			)
			oldURL, err := url.Parse(oldRawurl)
			if err != nil {
				k.logger.Error("failure in processing change to Service",
					zap.String("namespace", evt.Svc.Namespace),
					zap.String("name", evt.Svc.Name),
					zap.String("err", err.Error()),
				)
				return
			}
			oldGRPCServiceName := evt.OldSvc.Annotations[serviceNameAnnotationKey]
			oldVersion := evt.OldSvc.Annotations[serviceVersionAnnotationKey]
			k.Records.RemoveRecord(oldGRPCServiceName, oldVersion, oldURL)
			return
		}

		// gRPC service annotation was added to the Service
		gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]
		version := evt.Svc.Annotations[serviceVersionAnnotationKey]
		k.Records.SetRecord(gRPCServiceName, version, u)
	}
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
