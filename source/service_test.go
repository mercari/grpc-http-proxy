package source

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/mercari/grpc-http-proxy/errors"
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

func (f *fixture) newKubernetes() *Service {
	f.client = fake.NewSimpleClientset(f.objects...)
	k := NewService(f.client, "", f.logger)
	for _, s := range f.lister {
		k.informer.GetIndexer().Add(s)
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
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if svc != nil {
			return true, nil
		}
		return false, nil
	})
}

func waitForNoService(c kubernetes.Interface, namespace, name string) error {
	return wait.PollImmediate(1*time.Second, time.Minute*2, func() (bool, error) {
		svc, err := c.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if svc == nil {
			return true, nil
		}
		return false, nil
	})
}

type testCase struct {
	service string
	version string
	url     *url.URL
	code    int
}

func checkRecords(t *testing.T, k *Service, cases []testCase) {
	t.Helper()
	for _, tc := range cases {
		u, err := k.Resolve(tc.service, tc.version)
		if got, want := u, tc.url; !reflect.DeepEqual(got, want) {
			t.Errorf("%#v", err)
			t.Fatalf("got %v, want %v", got, want)
		}
		switch e := err.(type) {
		case *errors.ProxyError:
			if got, want := int(e.Code), tc.code; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		case nil:
			if got, want := -1, tc.code; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		default:
			t.Fatal("unexpected error type")
		}
	}

}

func TestServiceAdded(t *testing.T) {
	t.Run("create versioned services", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     parseURL(t, "foo-service.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
			{
				service: "Echo",
				version: "v2",
				url:     parseURL(t, "foo-service-v2.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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
				serviceVersionAnnotationKey: "v2",
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
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("create unversioned services", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "",
				url:     nil,
				code:    int(errors.VersionUndecidable),
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

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
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("create service with invalid ports", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     nil,
				code:    int(errors.ServiceUnresolvable),
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
			},
			[]core.ServicePort{},
		)
		_, err := f.client.Core().Services(fooV1.Namespace).Create(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)

		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})
}

func TestServiceDeleted(t *testing.T) {
	t.Run("delete versioned services", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     nil,
				code:    int(errors.ServiceUnresolvable),
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{

				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("delete unversioned services (one left)", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "",
				url:     parseURL(t, "foo-service-v2.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

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
		err = waitForService(f.client, fooV2.Namespace, fooV2.Name)
		if err != nil {
			t.Fatal(err)
		}

		// delete v1 of foo-service
		err = f.client.Core().Services(fooV1.Namespace).Delete(fooV1.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		err = waitForNoService(f.client, fooV1.Namespace, fooV1.Name)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("delete unversioned services (more than left)", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "",
				url:     nil,
				code:    int(errors.VersionUndecidable),
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v3 of foo-service
		fooV3 := newService(
			"foo-service-v3",
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
		_, err := f.client.Core().Services(fooV3.Namespace).Create(fooV3)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV3.Namespace, fooV3.Name)

		// create v4 of foo-service
		fooV4 := newService(
			"foo-service-v4",
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
		_, err = f.client.Core().Services(fooV4.Namespace).Create(fooV4)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV4.Namespace, fooV4.Name)

		// create v5 of foo-service
		fooV5 := newService(
			"foo-service-v5",
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
		_, err = f.client.Core().Services(fooV5.Namespace).Create(fooV5)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV5.Namespace, fooV5.Name)

		// delete v3 of foo-service
		err = f.client.Core().Services(fooV3.Namespace).Delete(fooV3.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		err = waitForNoService(f.client, fooV3.Namespace, fooV3.Name)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})
}

func TestServiceUpdated(t *testing.T) {
	t.Run("change name of service", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     nil,
				code:    int(errors.ServiceUnresolvable),
			},
			{
				service: "Ping",
				version: "v1",
				url:     parseURL(t, "foo-service.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooSvc := newService(
			"foo-service",
			"bar-ns",
			map[string]string{
				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("change version of service", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     nil,
				code:    int(errors.ServiceUnresolvable),
			},
			{
				service: "Echo",
				version: "v2",
				url:     parseURL(t, "foo-service.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooSvc := newService(
			"foo-service",
			"bar-ns",
			map[string]string{
				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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
		fooSvc.Annotations[serviceVersionAnnotationKey] = "v2"
		_, err = f.client.Core().Services(fooSvc.Namespace).Update(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("Add version to duplicate unversioned service", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "",
				url:     nil,
				code:    int(errors.VersionNotSpecified),
			},
			{
				service: "Echo",
				version: "v2",
				url:     parseURL(t, "foo-service-v2.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

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
		fooV2.Annotations[serviceVersionAnnotationKey] = "v2"
		_, err = f.client.Core().Services(fooV2.Namespace).Update(fooV2)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV2.Namespace, fooV2.Name)
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("change port (valid)", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     parseURL(t, "foo-service.bar-ns.svc.cluster.local:5001"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{
				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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

		// change port of foo-service v1
		fooV1.Spec.Ports = []core.ServicePort{
			{
				Name:     "grpc",
				Protocol: "TCP",
				Port:     5001,
			},
		}
		_, err = f.client.Core().Services(fooV1.Namespace).Update(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("change port (invalid)", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     nil,
				code:    int(errors.ServiceUnresolvable),
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooV1 := newService(
			"foo-service",
			"bar-ns",
			map[string]string{
				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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

		// change port of foo-service v1
		fooV1.Spec.Ports = []core.ServicePort{}
		_, err = f.client.Core().Services(fooV1.Namespace).Update(fooV1)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooV1.Namespace, fooV1.Name)
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("add gRPC service annotation to Service", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     parseURL(t, "foo-service.bar-ns.svc.cluster.local:5000"),
				code:    -1,
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooSvc := newService(
			"foo-service",
			"bar-ns",
			map[string]string{},
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

		// add gRPC service annotation
		fooSvc.Annotations[serviceNameAnnotationKey] = "Echo"
		// add version annotation
		fooSvc.Annotations[serviceVersionAnnotationKey] = "v1"
		_, err = f.client.Core().Services(fooSvc.Namespace).Update(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})

	t.Run("remove gRPC service annotation from Service", func(t *testing.T) {
		cases := []testCase{
			{
				service: "Echo",
				version: "v1",
				url:     nil,
				code:    int(errors.ServiceUnresolvable),
			},
		}
		f := newFixture(t)
		k := f.newKubernetes()
		stopCh := make(chan struct{})
		defer func() { stopCh <- struct{}{} }()
		k.Run(stopCh)
		time.Sleep(2 * time.Second)

		// create v1 of foo-service
		fooSvc := newService(
			"foo-service",
			"bar-ns",
			map[string]string{
				serviceNameAnnotationKey:    "Echo",
				serviceVersionAnnotationKey: "v1",
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

		// remove gRPC service annotation
		delete(fooSvc.Annotations, serviceNameAnnotationKey)
		// remove version annotation
		delete(fooSvc.Annotations, serviceVersionAnnotationKey)
		_, err = f.client.Core().Services(fooSvc.Namespace).Update(fooSvc)
		if err != nil {
			t.Fatal(err)
		}
		waitForService(f.client, fooSvc.Namespace, fooSvc.Name)
		time.Sleep(2 * time.Second)
		checkRecords(t, k, cases)
	})
}

func TestSelectPort(t *testing.T) {
	cases := []struct {
		name         string
		servicePorts []core.ServicePort
		port         int32
		ok           bool
	}{
		{
			name: "single",
			servicePorts: []core.ServicePort{
				{
					Name: "foo",
					Port: 5000,
				},
			},
			port: 5000,
			ok:   true,
		},
		{
			name: "one with prefix",
			servicePorts: []core.ServicePort{
				{
					Name: "grpc-foo",
					Port: 5000,
				},
				{
					Name: "bar",
					Port: 5001,
				},
			},
			port: 5000,
			ok:   true,
		},
		{
			name: "multiple without prefix",
			servicePorts: []core.ServicePort{
				{
					Name: "foo",
					Port: 5000,
				},
				{
					Name: "bar",
					Port: 5001,
				},
			},
			port: 0,
			ok:   false,
		},
		{
			name:         "none",
			servicePorts: []core.ServicePort{},
			port:         0,
			ok:           false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			port, ok := selectPort(tc.servicePorts)
			if got, want := port, tc.port; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
			if got, want := ok, tc.ok; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}
