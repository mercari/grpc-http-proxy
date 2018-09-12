package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Env struct {
	// LogLevel is INFO, DEBUG, or ERROR
	LogLevel string `envconfig:"LOG_LEVEL" default:"INFO"`

	// Port is the port number grpc-http-proxy will listen on
	Port int16 `envconfig:"PORT" default:"3000"`

	// Token is the access token
	Token string `envconfig:"TOKEN"`
}

func ReadFromEnv() (*Env, error) {
	var env Env
	if err := envconfig.Process("", &env); err != nil {
		return nil, errors.Wrap(err, "envconfig failed to read environment variables")
	}

	return &env, nil
}
