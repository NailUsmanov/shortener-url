#!/usr/bin/env zsh

# Конфигурация
LOG_FILE="test_results.log"
BINARY="cmd/shortener/shortener"
SOURCE_DIR="."
TEST_PORT=8080
TEST_BINARY="./shortenertestbeta"  # Укажите полный путь если нужно

# Очистка предыдущих логов
: > $LOG_FILE

# Функция для вывода с таймингом
log() {
  echo "[$(date +'%T')] $1" | tee -a $LOG_FILE
}

# Сборка приложения
build_app() {
  log "🔨 Собираю приложение..."
  go build -ldflags "-s -w -X main.buildVersion=1.0.0 -X main.buildDate=$(date +'%Y-%m-%d') -X main.buildCommit=$(git rev-parse --short HEAD)" \
    -o $BINARY cmd/shortener/main.go || {
    log "❌ Ошибка сборки!"
    exit 1
  }
  log "✅ Сборка успешна"
}

# Запуск тестов
run_tests() {
  local tests=(
    "Iteration1:-test.run=^TestIteration1$ -binary-path=$BINARY"
    "Iteration2:-test.run=^TestIteration2$ -source-path=$SOURCE_DIR"
    "Iteration3:-test.run=^TestIteration3$ -source-path=$SOURCE_DIR"
    "Iteration4:-test.run=^TestIteration4$ -binary-path=$BINARY -server-port=$TEST_PORT"
    "Iteration5:-test.run=^TestIteration5$ -binary-path=$BINARY -server-port=$TEST_PORT"
    "Iteration6:-test.run=^TestIteration6$ -binary-path=$BINARY -source-path=$SOURCE_DIR -server-port=$TEST_PORT"
  )

  for test in $tests; do
    local name=${test%%:*}
    local args=${test#*:}
    
    log "\n🔍 Запускаю $name с параметрами: $args"
    $TEST_BINARY -test.v $args >> $LOG_FILE 2>&1
    
    if grep -q "FAIL" $LOG_FILE; then
      log "❌ Тест $name не пройден!"
      grep -A 15 "FAIL" $LOG_FILE
      return 1
    else
      log "✅ $name пройден успешно"
    fi
  done
}

# Главная функция
main() {
  build_app
  run_tests || exit 1
  log "\n🎉 Все тесты успешно пройдены!"
  echo "Полные логи доступны в $LOG_FILE"
}

# Запуск
main