package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestNewConfig_Defaults(t *testing.T) {
	resetFlags()
	os.Clearenv()

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, "", cfg.SaveInFile)
}

func TestNewConfig_EnvOnly(t *testing.T) {
	resetFlags()
	os.Clearenv()

	os.Setenv("SERVER_ADDRESS", ":9000")
	os.Setenv("BASE_URL", "http://example.com")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/storage")

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":9000", cfg.RunAddr)
	assert.Equal(t, "http://example.com", cfg.BaseURL)
	assert.Equal(t, "/tmp/storage", cfg.SaveInFile)
}

func TestNewConfig_FlagsOverrideEnv(t *testing.T) {
	resetFlags()
	os.Clearenv()

	os.Setenv("SERVER_ADDRESS", ":3000")
	os.Setenv("BASE_URL", "http://env.com")
	os.Setenv("FILE_STORAGE_PATH", "/env/path")

	os.Args = []string{
		"cmd",
		"-a", ":4000",
		"-b", "http://flag.com",
		"-f", "/flag/path",
	}

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":4000", cfg.RunAddr)
	assert.Equal(t, "http://flag.com", cfg.BaseURL)
	assert.Equal(t, "/flag/path", cfg.SaveInFile)
}

func TestNewConfig_RunAddrWithoutColon(t *testing.T) {
	resetFlags()
	os.Clearenv()

	os.Args = []string{
		"cmd",
		"-a", "9090",
	}

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.RunAddr)
	assert.Equal(t, "http://localhost:9090", cfg.BaseURL)
}
