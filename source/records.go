package source

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/mercari/grpc-http-proxy/errors"
)

type versions map[string][]*url.URL

// Records contains mappings from a gRPC service to upstream hosts
// It holds one upstream for each service version
type Records struct {
	m     map[string]versions
	mutex sync.RWMutex
}

// NewRecords creates an empty mapping
func NewRecords() *Records {
	m := make(map[string]versions)
	return &Records{
		m:     m,
		mutex: sync.RWMutex{},
	}
}

// ClearRecords clears all mappings
func (r *Records) ClearRecords() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.m = make(map[string]versions)
}

// GetRecord gets a records of the specified (service, version) pair
func (r *Records) GetRecord(svc, version string) (*url.URL, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	vs, ok := r.m[svc]
	if !ok {
		return nil, &errors.Error{
			Code:    errors.ServiceUnresolvable,
			Message: fmt.Sprintf("The gRPC service %s is unresolvable", svc),
		}
	}
	if version == "" {
		if len(vs) != 1 {
			return nil, &errors.Error{
				Code: errors.VersionNotSpecified,
				Message: fmt.Sprintf("There are multiple version of the gRPC service %s available. "+
					"You must specify one", svc),
			}
		}
		for _, entries := range vs {
			if len(entries) != 1 {
				return nil, &errors.Error{
					Code: errors.VersionUndecidable,
					Message: fmt.Sprintf("Multiple possible backends found for the gRPC service %s. "+
						"Add annotations to distinguish versions", svc),
				}
			}
			return entries[0], nil // this returns the first (and only) ServiceURL
		}
	}
	entries, ok := vs[version]
	if !ok {
		return nil, &errors.Error{
			Code:    errors.ServiceUnresolvable,
			Message: fmt.Sprintf("Version %s of the gRPC service %s is unresolvable", version, svc),
		}
	}
	if len(entries) != 1 {
		return nil, &errors.Error{
			Code: errors.VersionUndecidable,
			Message: fmt.Sprintf("Multiple possible backends found for the gRPC service %s. "+
				"Add annotations to distinguish versions", svc),
		}
	}
	return entries[0], nil
}

// SetRecord sets the backend service URL for the specifiec (service, version) pair.
// When successful, true will be returned.
// This fails if the URL for the blank version ("") is to be overwritten, and invalidates that entry.
func (r *Records) SetRecord(svc, version string, u *url.URL) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.m[svc]; !ok {
		r.m[svc] = make(map[string][]*url.URL)
	}
	if r.m[svc][version] == nil {
		r.m[svc][version] = make([]*url.URL, 0)
	}
	r.m[svc][version] = append(r.m[svc][version], u)
	return true
}

// RemoveRecord removes a record of the specified (service, version) pair
func (r *Records) RemoveRecord(svc, version string, u *url.URL) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vs, ok := r.m[svc]
	if !ok {
		return
	}
	entries, ok := vs[version]
	if !ok {
		return
	}
	newEntries := make([]*url.URL, 0)
	for _, e := range entries {
		if e.String() != u.String() {
			newEntries = append(newEntries, e)
		}
	}
	vs[version] = newEntries
	if len(newEntries) == 0 {
		delete(vs, version)
	}
	if len(vs) == 0 {
		delete(r.m, svc)
	}
}

// IsServiceUnique checks if there is only one version of a service
func (r *Records) IsServiceUnique(svc string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	b := len(r.m[svc]) == 1
	return b
}

// RecordExists checks if a record exists
func (r *Records) RecordExists(svc, version string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	vs, ok := r.m[svc]
	if !ok {
		return false
	}
	entries, ok := vs[version]
	if !ok {
		return false
	}
	return len(entries) > 0
}
