package discoverer

import (
	"fmt"
	"net/url"
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

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/internal/records"
)

const (
	// TODO(tomoyat1) change annotation names
	serviceNameAnnotationKey    = "proto-service"
	backendVersionAnnotationKey = "backend-version"
)

// Kubernetes watches the Kubernetes API and updates records when there are changes to Service resources
type Kubernetes struct {
	records   *records.Records
	logger    *zap.Logger
	informer  cache.SharedIndexInformer
	namespace string
	lister    corelisters.ServiceLister
	queue     workqueue.RateLimitingInterface
}

// NewKubernetes creates a new Kubernetes Discoverer
func NewKubernetes(
	client clientset.Interface,
	namespace string,
	l *zap.Logger) *Kubernetes {

	opts := make([]informers.SharedInformerOption, 0)
	if namespace != "" {
		opts = append(opts, informers.WithNamespace(namespace))
	}
	infFactory := informers.NewSharedInformerFactoryWithOptions(client,
		time.Second, opts...)

	k := &Kubernetes{
		records:   records.NewRecords(),
		logger:    l,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
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
func (k *Kubernetes) Resolve(svc, version string) (proxy.ServiceURL, error) {
	r, err := k.records.GetRecord(svc, version)
	if err != nil {
		k.logger.Error("failed to resolve service",
			zap.String("service", svc),
			zap.String("version", version),
			zap.String("err", err.Error()))
		return nil, err
	}
	return r, nil
}

// Run starts the Kubernetes controller
func (k *Kubernetes) Run(stopCh <-chan struct{}) {
	go k.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh,
		k.informer.HasSynced,
	) {
		k.logger.Error("timed out waiting for caches to sync")
	}
	go wait.Until(k.runWorker, time.Second, stopCh)
}

func (k *Kubernetes) runWorker() {
	for k.processNextItem() {
	}
}

func (k *Kubernetes) processNextItem() bool {
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

func (k *Kubernetes) eventHandler(evt Event) {
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
	switch evt.EventType {
	case createEvent:
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Info("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}
		gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]

		if metav1.HasAnnotation(evt.Svc.ObjectMeta, backendVersionAnnotationKey) {
			version := evt.Svc.Annotations[backendVersionAnnotationKey]
			k.records.SetRecord(gRPCServiceName, version, u)
			return
		}
		k.records.SetRecord(gRPCServiceName, "", u)
	case deleteEvent:
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Info("skipping service because of no annotation",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
			)
			return
		}
		gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]

		if metav1.HasAnnotation(evt.Svc.ObjectMeta, backendVersionAnnotationKey) {
			version := evt.Svc.Annotations[backendVersionAnnotationKey]
			k.records.RemoveRecord(gRPCServiceName, version)
		} else {
			// recreate entire record table to prevent avoid edge cases
			k.recreateRecordTable(evt)
		}
	case updateEvent:
		// Service versions before and after update do not have annotations
		// Skip service and return
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) &&
			!metav1.HasAnnotation(evt.OldSvc.ObjectMeta, serviceNameAnnotationKey) {
			k.logger.Info("skipping service because of no annotation",
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
			version := evt.Svc.Annotations[backendVersionAnnotationKey]
			oldVersion := evt.OldSvc.Annotations[backendVersionAnnotationKey]

			if oldGRPCServiceName != gRPCServiceName {
				// gRPC service name was changed
				if oldVersion != "" && version != "" {
					// safe to remove and add, since both old and new are versioned
					k.records.RemoveRecord(oldGRPCServiceName, oldVersion)
					k.records.SetRecord(gRPCServiceName, version, u)
					return
				}
				// recreate record table to avoid edge cases around empty versions
				k.recreateRecordTable(evt)
				return
			}

			if version != oldVersion {
				// version annotation was changed
				if oldVersion != "" && version != "" {
					// safe to remove and add, since both old and new are versioned
					k.records.RemoveRecord(oldGRPCServiceName, oldVersion)
					k.records.SetRecord(gRPCServiceName, version, u)
					return
				}
				// recreate record table to avoid edge cases around empty versions
				k.recreateRecordTable(evt)
				return
			}

			if !k.records.RecordExists(gRPCServiceName, version) {
				// Record is missing, so add it
				k.records.SetRecord(gRPCServiceName, version, u)
				return
			}

			// do nothing, since no annotations were changed
			return
		}

		// gRPC service annotation was removed from the Service
		if !metav1.HasAnnotation(evt.Svc.ObjectMeta, serviceNameAnnotationKey) {
			oldGRPCServiceName := evt.OldSvc.Annotations[serviceNameAnnotationKey]
			oldVersion := evt.OldSvc.Annotations[backendVersionAnnotationKey]
			k.records.RemoveRecord(oldGRPCServiceName, oldVersion)
			return
		}

		// gRPC service annotation was added to the Service
		gRPCServiceName := evt.Svc.Annotations[serviceNameAnnotationKey]
		version := evt.Svc.Annotations[backendVersionAnnotationKey]
		k.records.SetRecord(gRPCServiceName, version, u)
	}
}

func (k *Kubernetes) recreateRecordTable(evt Event) {
	// The following logic recreates the mapping between gRPC services and Kubernetes Services
	// every time there is a change to a service somewhere in the cluster.
	// This does not scale well in clusters with large amounts of Services, so it is only used
	// in some places to avoid edge cases.
	objs := k.informer.GetStore().List()
	svcs := make([]*core.Service, 0)
	for _, o := range objs {
		s, ok := o.(*core.Service)
		if !ok {
			k.logger.Error(fmt.Sprintf("invalid object in Store; got %T want *core.Service", o))
			continue
		}
		if metav1.HasAnnotation(s.ObjectMeta, serviceNameAnnotationKey) {
			svcs = append(svcs, s)
		}
	}
	k.records.ClearRecords()
	for _, s := range svcs {
		gRPCServiceName := s.Annotations[serviceNameAnnotationKey]
		rawurl := fmt.Sprintf("%s.%s.svc.cluster.local", s.Name, s.Namespace)
		u, err := url.Parse(rawurl)
		if err != nil {
			k.logger.Error("failure in processing change to Service",
				zap.String("namespace", evt.Svc.Namespace),
				zap.String("name", evt.Svc.Name),
				zap.String("err", err.Error()),
			)
			return
		}
		if metav1.HasAnnotation(s.ObjectMeta, backendVersionAnnotationKey) {
			version := s.Annotations[backendVersionAnnotationKey]
			k.records.SetRecord(gRPCServiceName, version, u)
		} else {
			k.records.SetRecord(gRPCServiceName, "", u)
		}
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
