package discoverer

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/mercari/grpc-http-proxy"
)

type versions map[string]entry

type entry struct {
	decidable bool
	url       proxy.ServiceURL
}

type records struct {
	m     map[string]versions
	mutex sync.RWMutex
}

func serviceNotFound(svc string) *proxy.Error {
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

func NewRecords() *records {
	m := make(map[string]versions)

	return &records{
		m:     m,
		mutex: sync.RWMutex{},
	}
}

func NewRecordsFromYAML(yamlFile string) (*records, error) {
	r := NewRecords()
	rawMapping := make(map[string]map[string]string)
	f, err := os.Open(yamlFile)
	if err != nil {
		return nil, err
	}
	d := yaml.NewDecoder(f)
	err = d.Decode(rawMapping)
	if err != nil {
		return nil, err
	}
	for service, versions := range rawMapping {
		for version, rawurl := range versions {
			u, err := url.Parse(rawurl)
			if err != nil {
				return nil, err
			}
			r.SetRecord(service, version, u)
		}
	}
	return r, nil
}

func (r *records) ClearRecords() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.m = make(map[string]versions)
}

func (r records) GetRecord(svc, version string) (proxy.ServiceURL, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	vs, ok := r.m[svc]
	if !ok {
		return nil, serviceNotFound(svc)
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
func (r records) SetRecord(svc, version string, url proxy.ServiceURL) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.m[svc]; !ok {
		r.m[svc] = make(map[string]entry)
	}
	if _, ok := r.m[svc][version]; ok && version == "" {
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

func (r *records) RemoveRecord(svc, version string) {
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

func (r *records) IsServiceUnique(svc string) bool {
	r.mutex.RLock()
	b := len(r.m[svc]) == 1
	r.mutex.RUnlock()
	return b
}
