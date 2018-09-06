package records

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/mercari/grpc-http-proxy"
)

type versions map[string]entry

type entry struct {
	decidable bool
	url       proxy.ServiceURL
}

type Records struct {
	m     map[string]versions
	mutex sync.RWMutex
}

func serviceUnresolvable(svc string) *proxy.Error {
	return &proxy.Error{
		Code:    proxy.ServiceUnresolvable,
		Message: fmt.Sprintf("The gRPC service %s is unresolvable", svc),
	}
}

func versionNotFound(svc, version string) *proxy.Error {
	return &proxy.Error{
		Code:    proxy.ServiceUnresolvable,
		Message: fmt.Sprintf("Version %s of the gRPC service %s is unresolvable", version, svc),
	}
}

func versionNotSpecified(svc string) *proxy.Error {
	return &proxy.Error{
		Code: proxy.VersionNotSpecified,
		Message: fmt.Sprintf("There are multiple version of the gRPC service %s available. "+
			"You must specify one", svc),
	}
}

func versionUndecidable(svc string) *proxy.Error {
	return &proxy.Error{
		Code: proxy.VersionUndecidable,
		Message: fmt.Sprintf("Multiple possible backends found for the gRPC service %s. "+
			"Add annotations to distinguish versions", svc),
	}
}

func NewRecords() *Records {
	m := make(map[string]versions)
	return &Records{
		m:     m,
		mutex: sync.RWMutex{},
	}
}

func (r *Records) ClearRecords() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.m = make(map[string]versions)
}

func (r *Records) GetRecord(svc, version string) (proxy.ServiceURL, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	vs, ok := r.m[svc]
	if !ok {
		return nil, serviceUnresolvable(svc)
	}
	if version == "" {
		if len(vs) != 1 {
			return nil, versionNotSpecified(svc)
		}
		for _, e := range vs {
			if !e.decidable {
				return nil, versionUndecidable(svc)
			}
			return e.url, nil // this returns the first (and only) ServiceURL
		}
	}
	e, ok := vs[version]
	if !ok {
		return nil, versionNotFound(svc, version)
	}
	if !e.decidable {
		return nil, versionUndecidable(svc)
	}
	return e.url, nil
}

// SetRecord sets the backend service URL for the specifiec (service, version) pair.
// When successful, true will be returned.
// This fails if the URL for the blank version ("") is to be overwritten, and invalidates that entry.
func (r *Records) SetRecord(svc, version string, url proxy.ServiceURL) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.m[svc]; !ok {
		r.m[svc] = make(map[string]entry)
	}
	if _, ok := r.m[svc][version]; ok && version == "" {
		// if there are multiple backends for a given gRPC service,
		// the blank version for it becomes undecidable.
		r.m[svc][version] = entry{
			decidable: false,
		}
		return false
	}
	r.m[svc][version] = entry{
		decidable: true,
		url:       url,
	}
	return true
}

func (r *Records) RemoveRecord(svc, version string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vs, ok := r.m[svc]
	if !ok {
		return
	}
	delete(vs, version)
	if len(vs) < 1 {
		delete(r.m, svc)
	}
}

func (r *Records) IsServiceUnique(svc string) bool {
	r.mutex.RLock()
	b := len(r.m[svc]) == 1
	r.mutex.RUnlock()
	return b
}

func (r *Records) RecordExists(svc, version string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	vs, ok := r.m[svc]
	if !ok {
		return false
	}
	_, ok = vs[version]
	return ok
}

// Equals checks record table equality. This is useful for writing tests
func (r *Records) Equals(r2 *Records) bool {
	r.mutex.RLock()
	r2.mutex.RLock()
	defer r.mutex.RUnlock()
	defer r2.mutex.RUnlock()
	return reflect.DeepEqual(r.m, r2.m)
}
