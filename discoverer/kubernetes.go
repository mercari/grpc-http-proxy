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
)

const (
	// TODO(tomoyat1) change annotation names
	serviceNameAnnotationKey    = "proto-service"
	backendVersionAnnotationKey = "backend-version"
)

// Kubernetes watches the Kubernetes api and updates records when there are changes to Service resources
type Kubernetes struct {
	*records
	logger    *zap.Logger
	informer  cache.SharedIndexInformer
	namespace string
	lister    corelisters.ServiceLister
	*WorkQueue
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
		records:   NewRecords(),
		logger:    l,
		WorkQueue: NewWorkQueue(),
		namespace: namespace,
	}
	serviceInformer := infFactory.Core().V1().Services()
	k.informer = serviceInformer.Informer()
	k.lister = serviceInformer.Lister()
	eventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc, ok := obj.(*core.Service)
			if !ok {
				return
			}
			k.enqueue(Event{
				EventType: createEvent,
				Meta:      &svc.ObjectMeta,
			})
			return
		},
		DeleteFunc: func(obj interface{}) {
			svc, ok := obj.(*core.Service)
			if !ok {
				return
			}
			k.enqueue(Event{
				EventType: deleteEvent,
				Meta:      &svc.ObjectMeta,
			})
			return
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newSvc, ok := newObj.(*core.Service)
			if !ok {
				return
			}
			k.enqueue(Event{
				EventType: updateEvent,
				Meta:      &newSvc.ObjectMeta,
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
	// The following logic recreates the mapping between gRPC services and Kubernetes Services
	// every time there is a change to a service somewhere in the cluster.
	// This does not scale well in clusters with large amounts of Services, and needs a fix.
	objs := k.informer.GetStore().List()
	svcs := make([]*core.Service, 0)
	for _, o := range objs {
		s, ok := o.(*core.Service)
		if !ok {
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
				zap.String("namespace", evt.Meta.Namespace),
				zap.String("name", evt.Meta.Name),
				zap.String("err", err.Error()),
			)
			return
		}
		if metav1.HasAnnotation(s.ObjectMeta, backendVersionAnnotationKey) {
			version := s.Annotations[backendVersionAnnotationKey]
			k.SetRecord(gRPCServiceName, version, u)
		} else {
			k.SetRecord(gRPCServiceName, "", u)
		}
	}
}

// Event is an change event to a Kubernetes Service
type Event struct {
	EventType
	Meta *metav1.ObjectMeta
}

// EventType is the type of an event
type EventType string

const (
	createEvent EventType = "CREATE"
	updateEvent EventType = "UPDATE"
	deleteEvent EventType = "DELETE"
)

// WorkQueue has a queue of Events in need of processing
type WorkQueue struct {
	queue workqueue.RateLimitingInterface
}

// NewWorkQueue creates a new WorkQueue
func NewWorkQueue() *WorkQueue {
	return &WorkQueue{
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
	}
}

func (q *WorkQueue) enqueue(evt Event) {
	q.queue.AddRateLimited(evt)
}
