package config

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Парсим флаги один раз для всех тестов
	flag.Parse()
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	// Сохраняем оригинальные значения флагов
	oldRunAddr := *flagRunAddr
	oldBaseURL := *flagBaseURL
	oldSaveInFile := *flagSaveInFile
	defer func() {
		*flagRunAddr = oldRunAddr
		*flagBaseURL = oldBaseURL
		*flagSaveInFile = oldSaveInFile
	}()

	t.Run("Default values", func(t *testing.T) {
		os.Clearenv()
		*flagRunAddr = ""
		*flagBaseURL = ""
		*flagSaveInFile = ""

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.RunAddr != ":8080" {
			t.Errorf("Expected RunAddr :8080, got %s", cfg.RunAddr)
		}
		if cfg.BaseURL != "http://localhost:8080" {
			t.Errorf("Expected BaseURL http://localhost:8080, got %s", cfg.BaseURL)
		}
		if cfg.SaveInFile != "" {
			t.Errorf("Expected empty SaveInFile, got %s", cfg.SaveInFile)
		}
	})

	t.Run("Environment variables", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("SERVER_ADDRESS", ":9090")
		os.Setenv("BASE_URL", "https://example.com")
		os.Setenv("FILE_STORAGE_PATH", "/tmp/data.json")
		defer os.Clearenv()

		*flagRunAddr = ""
		*flagBaseURL = ""
		*flagSaveInFile = ""

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.RunAddr != ":9090" {
			t.Errorf("Expected RunAddr :9090, got %s", cfg.RunAddr)
		}
		if cfg.BaseURL != "https://example.com" {
			t.Errorf("Expected BaseURL https://example.com, got %s", cfg.BaseURL)
		}
		if cfg.SaveInFile != "/tmp/data.json" {
			t.Errorf("Expected SaveInFile /tmp/data.json, got %s", cfg.SaveInFile)
		}
	})

	t.Run("Command line flags", func(t *testing.T) {
		os.Clearenv()
		*flagRunAddr = ":7070"
		*flagBaseURL = "http://flag"
		*flagSaveInFile = "flag.json"

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.RunAddr != ":7070" {
			t.Errorf("Expected RunAddr :7070, got %s", cfg.RunAddr)
		}
		if cfg.BaseURL != "http://flag" {
			t.Errorf("Expected BaseURL http://flag, got %s", cfg.BaseURL)
		}
		if cfg.SaveInFile != "flag.json" {
			t.Errorf("Expected SaveInFile flag.json, got %s", cfg.SaveInFile)
		}
	})
}
