package config

import (
	"os"
	"testing"
)

func TestReadFromEnv(t *testing.T) {
	pairs := map[string]string{
		"LOG_LEVEL": "ERROR",
		"PORT":      "8000",
		"TOKEN":     "foo",
	}

	reset := setEnvs(t, pairs)
	defer reset()

	env, err := ReadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := env.LogLevel, "ERROR"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got, want := env.Port, int16(8000); got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
	if got, want := env.Token, "foo"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestReadFromEnvLogLevelDefault(t *testing.T) {
	pairs := map[string]string{
		"PORT":  "8000",
		"TOKEN": "foo",
	}

	reset := setEnvs(t, pairs)
	defer reset()

	env, err := ReadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := env.LogLevel, "INFO"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got, want := env.Port, int16(8000); got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

func TestReadFromEnvPortDefault(t *testing.T) {
	pairs := map[string]string{
		"LOG_LEVEL": "ERROR",
		"TOKEN":     "foo",
	}

	reset := setEnvs(t, pairs)
	defer reset()

	env, err := ReadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := env.LogLevel, "ERROR"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got, want := env.Port, int16(3000); got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
	if got, want := env.Token, "foo"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestReadFromEnvTokenDefault(t *testing.T) {
	pairs := map[string]string{
		"LOG_LEVEL": "ERROR",
		"PORT":      "3000",
	}

	reset := setEnvs(t, pairs)
	defer reset()

	env, err := ReadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := env.Token, ""; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func setEnv(t *testing.T, key, value string) func() {
	original := os.Getenv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
	return func() {
		if original == "" {
			os.Unsetenv(key)
		} else {
			if err := os.Setenv(key, original); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func unsetEnv(t *testing.T, key string) func() {
	original := os.Getenv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	return func() {
		if original == "" {
			os.Unsetenv(key)
		} else {
			if err := os.Setenv(key, original); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func setEnvs(t *testing.T, envs map[string]string) func() {
	var resetFuncs []func()
	t.Helper()
	for k, v := range envs {
		r := setEnv(t, k, v)
		resetFuncs = append(resetFuncs, r)
	}
	return func() {
		for _, f := range resetFuncs {
			f()
		}
	}
}
