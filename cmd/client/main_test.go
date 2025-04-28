package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestClient(t *testing.T) {
	// 1. Создаем мок сервера
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://short.url/abc123"))
	}))
	defer ts.Close()

	// 2. Подменяем глобальные переменные
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	oldEndpoint := endpoint

	// 3. Настраиваем pipes для ввода/вывода
	stdinR, stdinW, _ := os.Pipe()
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdin = stdinR
	os.Stdout = stdoutW
	endpoint = ts.URL

	// 4. Записываем тестовый ввод
	go func() {
		defer stdinW.Close()
		stdinW.Write([]byte("https://example.com\n"))
	}()

	// 5. Запускаем main с синхронизацией
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			// Восстанавливаем оригинальные значения
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			endpoint = oldEndpoint
			stdoutW.Close()
		}()
		main()
	}()

	// 6. Читаем вывод
	var buf bytes.Buffer
	io.Copy(&buf, stdoutR)

	// 7. Ждем завершения
	wg.Wait()

	// 8. Проверяем результат
	output := buf.String()
	if !strings.Contains(output, "201") {
		t.Errorf("Expected status code 201 in output, got: %s", output)
	}
	if !strings.Contains(output, "http://short.url/abc123") {
		t.Errorf("Expected short URL in output, got: %s", output)
	}
}
