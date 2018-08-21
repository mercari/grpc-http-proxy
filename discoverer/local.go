package discoverer

import (
	"go.uber.org/zap"

	"github.com/mercari/grpc-http-proxy"
)

type Local struct {
	*records
	logger *zap.Logger
}

func NewLocal(l *zap.Logger, mappingFile string) Discoverer {
	local := &Local{
		logger: l,
	}
	r, err := NewRecordsFromYAML(mappingFile)
	if err != nil {
		local.logger.Error("failed to initialize records from yaml",
			zap.String("err", err.Error()))
		local.records = NewRecords()
		return local
	}
	local.records = r
	return local
}

func (l *Local) Resolve(svc, version string) (proxy.ServiceURL, error) {
	r, err := l.records.GetRecord(svc, version)
	if err != nil {
		l.logger.Error("failed to resolve service",
			zap.String("service", svc),
			zap.String("version", version),
			zap.String("err", err.Error()))
		return nil, err
	}
	return r, nil
}
