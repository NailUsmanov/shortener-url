package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMain_Success(t *testing.T) {
	// 1. Сохраняем оригинальные потоки
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	// 2. Настраиваем stdin через pipe
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Write([]byte("https://example.com\n"))
	w.Close()

	// 3. Перехватываем stdout через pipe
	outR, outW, _ := os.Pipe()
	os.Stdout = outW

	// 4. Создаем мок сервера
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://short.url/abc123"))
	}))
	defer ts.Close()

	// 5. Запускаем main в горутине (так как pipe блокирует)
	go func() {
		defer outW.Close()
		main()
	}()

	// 6. Читаем вывод
	var buf bytes.Buffer
	io.Copy(&buf, outR)

	// 7. Проверяем вывод
	if !strings.Contains(buf.String(), "Статус-код 201 Created") {
		t.Error("Expected status 201 in output")
	}
	if !strings.Contains(buf.String(), "http://short.url/abc123") {
		t.Error("Expected short URL in output")
	}
}

func TestMain_ServerError(t *testing.T) {
	// Аналогичная настройка как в TestMain_Success
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Write([]byte("https://example.com\n"))
	w.Close()

	outR, outW, _ := os.Pipe()
	os.Stdout = outW

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	go func() {
		defer outW.Close()
		main()
	}()

	var buf bytes.Buffer
	io.Copy(&buf, outR)

	if !strings.Contains(buf.String(), "500") {
		t.Error("Expected 500 error in output")
	}
}
