package source

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/mercari/grpc-http-proxy/errors"
)

type versions map[string][]*url.URL

// Records contains mappings from a gRPC service to upstream hosts
// It holds one upstream for each service version
type Records struct {
	M         map[string]versions `json:"grpc_service"`
	recordsMu sync.RWMutex
}

// NewRecords creates an empty mapping
func NewRecords() *Records {
	m := make(map[string]versions)
	return &Records{
		M:         m,
		recordsMu: sync.RWMutex{},
	}
}

func (r *Records) ToJSON() []byte {
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()

	j, err := json.Marshal(r)
	if err != nil {
		return []byte(err.Error())
	}

	return j
}

// ClearRecords clears all mappings
func (r *Records) ClearRecords() {
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()
	r.M = make(map[string]versions)
}

// GetRecord gets a records of the specified (service, version) pair
func (r *Records) GetRecord(svc, version string) (*url.URL, error) {
	r.recordsMu.RLock()
	defer r.recordsMu.RUnlock()
	vs, ok := r.M[svc]
	if !ok {
		return nil, &errors.ProxyError{
			Code:    errors.ServiceUnresolvable,
			Message: fmt.Sprintf("The gRPC service %s is unresolvable", svc),
		}
	}
	if version == "" {
		if len(vs) != 1 {
			return nil, &errors.ProxyError{
				Code: errors.VersionNotSpecified,
				Message: fmt.Sprintf("There are multiple version of the gRPC service %s available. "+
					"You must specify one", svc),
			}
		}
		for _, entries := range vs {
			if len(entries) != 1 {
				return nil, &errors.ProxyError{
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
		return nil, &errors.ProxyError{
			Code:    errors.ServiceUnresolvable,
			Message: fmt.Sprintf("Version %s of the gRPC service %s is unresolvable", version, svc),
		}
	}
	if len(entries) != 1 {
		return nil, &errors.ProxyError{
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
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()
	if _, ok := r.M[svc]; !ok {
		r.M[svc] = make(map[string][]*url.URL)
	}
	if r.M[svc][version] == nil {
		r.M[svc][version] = make([]*url.URL, 0)
	}
	r.M[svc][version] = append(r.M[svc][version], u)
	return true
}

// RemoveRecord removes a record of the specified (service, version) pair
func (r *Records) RemoveRecord(svc, version string, u *url.URL) {
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()

	vs, ok := r.M[svc]
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
		delete(r.M, svc)
	}
}

// IsServiceUnique checks if there is only one version of a service
func (r *Records) IsServiceUnique(svc string) bool {
	r.recordsMu.RLock()
	defer r.recordsMu.RUnlock()
	b := len(r.M[svc]) == 1
	return b
}

// RecordExists checks if a record exists
func (r *Records) RecordExists(svc, version string) bool {
	r.recordsMu.RLock()
	defer r.recordsMu.RUnlock()
	vs, ok := r.M[svc]
	if !ok {
		return false
	}
	entries, ok := vs[version]
	if !ok {
		return false
	}
	return len(entries) > 0
}
