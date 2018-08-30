package discoverer

import (
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type fixture struct {
	t       *testing.T
	client  *fake.Clientset
	objects []runtime.Object
	logger  *zap.Logger
	logs    *observer.ObservedLogs

	lister []*core.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.lister = []*core.Service{}
	c, logs := observer.New(zapcore.InfoLevel)
	f.logger = zap.New(c)
	f.logs = logs
	return f
}

func (f *fixture) newKubernetes() *Kubernetes {
	f.client = fake.NewSimpleClientset(f.objects...)
	k := NewKubernetes(f.client, "", f.logger)
	for _, s := range f.lister {
		k.informer.GetIndexer().Add(s)
	}
	for _, s := range f.objects {
		f.objects = append(f.objects, s)
	}
	return k
}

func newService(name, namespace string, annotations map[string]string, ports []core.ServicePort) *core.Service {
	return &core.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: core.ServiceSpec{
			Ports: ports,
		},
	}
}

func waitForService(c kubernetes.Interface, namespace, name string) error {
	return wait.PollImmediate(1*time.Second, time.Minute*2, func() (bool, error) {
		svc, err := c.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if svc != nil {
			return true, nil
		}
		return false, nil
	})
}

func TestServiceAdded(t *testing.T) {
	t.Run("create versioned services", func(t *testing.T) {
		m := map[string]versions{
			"Echo": {
				"v1": entry{
					decidable: true,
					url:       parseUrl("foo-service.bar-ns.svc.cluster.local", t),
				},
				"v2": entry{
					decidable: true,
					url:       parseUrl("foo-service-v2.bar-ns.svc.cluster.local", t),
				},
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				backendVersionAnnotationKey: "v1",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooV1.Namespace).Create(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)

		// create v2 of foo-service
		fooV2 := newService(
			"foo-service-v2",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				backendVersionAnnotationKey: "v2",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err = f.client.Core().Services(fooV2.Namespace).Create(fooV2)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV2.Namespace, fooV2.Name)
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("create unversioned services", func(t *testing.T) {
		m := map[string]versions{
			"Echo": {
				"": entry{
					decidable: false,
				},
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey: "Echo",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooV1.Namespace).Create(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)

		// create v2 of foo-service
		fooV2 := newService(
			"foo-service-v2",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey: "Echo",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err = f.client.Core().Services(fooV2.Namespace).Create(fooV2)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV2.Namespace, fooV2.Name)
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestServiceDeleted(t *testing.T) {
	t.Run("delete versioned services", func(t *testing.T) {
		m := map[string]versions{}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				backendVersionAnnotationKey: "v1",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooV1.Namespace).Create(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)

		// delete v1 of foo-service
		err = f.client.Core().Services(fooV1.Namespace).Delete(fooV1.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("delete unversioned services", func(t *testing.T) {
		m := map[string]versions{
			"Echo": {
				"": entry{
					decidable: true,
					url:       parseUrl("foo-service-v2.bar-ns.svc.cluster.local", t),
				},
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey: "Echo",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooV1.Namespace).Create(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)

		// create v2 of foo-service
		fooV2 := newService(
			"foo-service-v2",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey: "Echo",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err = f.client.Core().Services(fooV2.Namespace).Create(fooV2)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV2.Namespace, fooV2.Name)

		// delete v1 of foo-service
		err = f.client.Core().Services(fooV1.Namespace).Delete(fooV1.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestServiceUpdated(t *testing.T) {
	t.Run("change name of service", func(t *testing.T) {
		m := map[string]versions{
			"Ping": {
				"v1": entry{
					true,
					parseUrl("foo-service.bar-ns.svc.cluster.local", t),
				},
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooSvc := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				backendVersionAnnotationKey: "v1",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooSvc.Namespace).Create(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)

		// change foo-service name to Ping
		fooSvc.Annotations[serviceNameAnnotationKey] = "Ping"
		_, err = f.client.Core().Services(fooSvc.Namespace).Update(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("change version of service", func(t *testing.T) {
		m := map[string]versions{
			"Echo": {
				"v2": entry{
					true,
					parseUrl("foo-service.bar-ns.svc.cluster.local", t),
				},
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooSvc := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				backendVersionAnnotationKey: "v1",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooSvc.Namespace).Create(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)

		// change foo-service version to v2
		fooSvc.Annotations[backendVersionAnnotationKey] = "v2"
		_, err = f.client.Core().Services(fooSvc.Namespace).Update(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("Add version to duplicate unversioned service", func(t *testing.T) {
		m := map[string]versions{
			"Echo": {
				"": entry{
					true,
					parseUrl("foo-service.bar-ns.svc.cluster.local", t),
				},
				"v2": entry{
					true,
					parseUrl("foo-service-v2.bar-ns.svc.cluster.local", t),
				},
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		k.Run(stopCh)
		time.Sleep(time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey: "Echo",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err := f.client.Core().Services(fooV1.Namespace).Create(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)

		// create v2 of foo-service
		fooV2 := newService(
			"foo-service-v2",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey: "Echo",
			},
			[]core.ServicePort{
				{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     5000,
				},
			},
		)
		_, err = f.client.Core().Services(fooV2.Namespace).Create(fooV2)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV2.Namespace, fooV2.Name)

		// add version annotation to v2 of foo-service
		fooV2.Annotations[backendVersionAnnotationKey] = "v2"
		_, err = f.client.Core().Services(fooV2.Namespace).Update(fooV2)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV2.Namespace, fooV2.Name)
		time.Sleep(1 * time.Second)
		if got, want := k.records.m, m; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}
