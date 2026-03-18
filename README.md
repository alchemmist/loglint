# loglint

Go-линтер для проверки лог-записей, совместимый с [golangci-lint](https://golangci-lint.run/).

Анализирует вызовы логгеров в коде и проверяет сообщения на соответствие правилам.

## Правила

| # | Правило | Описание |
|---|---------|----------|
| 1 | **lowercase_start** | Лог-сообщения должны начинаться со строчной буквы |
| 2 | **english_only** | Лог-сообщения должны быть только на английском языке |
| 3 | **no_special_chars** | Лог-сообщения не должны содержать спецсимволы, эмодзи или повторяющуюся пунктуацию |
| 4 | **no_sensitive_data** | Лог-сообщения не должны содержать потенциально чувствительные данные |

## Поддерживаемые логгеры

- `log/slog` — стандартный структурированный логгер Go
- `go.uber.org/zap` — Logger и SugaredLogger
- `log` — стандартный пакет логирования

## Установка

### Как standalone-инструмент

```bash
go install github.com/loglint/loglint/cmd/loglint@latest
```

### Из исходного кода

```bash
git clone https://github.com/loglint/loglint.git
cd loglint
make build
```

## Сборка

```bash
# Скачать зависимости
make deps

# Собрать бинарник
make build

# Собрать плагин для golangci-lint
make plugin

# Запустить тесты
make test

# Запустить тесты с покрытием
make test-cover

# Проверить форматирование и go vet
make lint

# Всё вместе
make all
```

## Использование

### Standalone

```bash
# Проверить текущий проект
loglint ./...

# Проверить конкретный пакет
loglint ./internal/server/...

# С указанием конфигурационного файла
loglint -config .loglint.yml ./...
```

### Как плагин golangci-lint

1. Соберите плагин:

```bash
make plugin
```

2. Настройте `.golangci.yml`:

```yaml
linters-settings:
  custom:
    loglint:
      path: ./loglint.so
      description: "Linter for checking log messages"
      original-url: github.com/loglint/loglint

linters:
  enable:
    - loglint
```

3. Запустите:

```bash
golangci-lint run
```

## Конфигурация

Создайте файл `.loglint.yml` в корне проекта:

```yaml
rules:
  # Правило 1: Сообщения должны начинаться со строчной буквы
  lowercase_start: true
  # Правило 2: Сообщения только на английском языке
  english_only: true
  # Правило 3: Без спецсимволов и эмодзи
  no_special_chars: true
  # Правило 4: Без чувствительных данных
  no_sensitive_data: true

patterns:
  # Кастомные ключевые слова для обнаружения чувствительных данных
  sensitive_keywords:
    - password
    - secret
    - token
    - api_key
    - private_key
    - my_custom_secret  # свои собственные паттерны
```

Поддерживаемые форматы: `.loglint.yml`, `.loglint.yaml`, `.loglint.json`.

Линтер ищет конфигурационный файл в текущей директории и родительских директориях.

## Примеры

### Правило 1: Строчная буква в начале

```go
// Неправильно
slog.Info("Starting server on port 8080")
slog.Error("Failed to connect to database")

// Правильно
slog.Info("starting server on port 8080")
slog.Error("failed to connect to database")
```

### Правило 2: Только английский язык

```go
// Неправильно
log.Info("запуск сервера")
log.Error("ошибка подключения к базе данных")

// Правильно
log.Info("starting server")
log.Error("failed to connect to database")
```

### Правило 3: Без спецсимволов и эмодзи

```go
// Неправильно
log.Info("server started! 🚀")
log.Error("connection failed!!!")
log.Warn("warning: something went wrong...")

// Правильно
log.Info("server started")
log.Error("connection failed")
log.Warn("something went wrong")
```

### Правило 4: Без чувствительных данных

```go
// Неправильно
log.Info("user password: " + password)
log.Debug("api_key=" + apiKey)
log.Info("token: " + token)

// Правильно
log.Info("user authenticated successfully")
log.Debug("api request completed")
log.Info("token validated")
```

### Автоисправление (SuggestedFixes)

Линтер предоставляет автоматические исправления для некоторых правил:

- **Правило 1**: Автоматически заменяет первую заглавную букву на строчную
- **Правило 3**: Автоматически удаляет спецсимволы, эмодзи и лишнюю повторяющуюся пунктуацию

Для применения исправлений используйте флаг `-fix`:

```bash
loglint -fix ./...
```

## Структура проекта

```
loglint/
├── cmd/
│   └── loglint/
│       └── main.go           # Точка входа standalone-инструмента
├── pkg/
│   └── analyzer/
│       ├── analyzer.go        # Основной анализатор
│       ├── rules.go           # Реализация правил проверки
│       ├── config.go          # Загрузка и парсинг конфигурации
│       └── analyzer_test.go   # Тесты
├── plugin/
│   └── plugin.go              # Плагин для golangci-lint
├── testdata/
│   └── src/                   # Тестовые данные
├── .github/
│   └── workflows/
│       └── ci.yml             # CI/CD pipeline
├── .golangci.yml              # Конфигурация golangci-lint
├── .loglint.yml               # Пример конфигурации линтера
├── .gitignore
├── go.mod
├── Makefile
└── README.md
```

## Технические требования

- Go 1.22+
- `golang.org/x/tools/go/analysis` — фреймворк для создания анализаторов

## CI/CD

Проект включает GitHub Actions workflow (`.github/workflows/ci.yml`):

- **Test**: запуск тестов с race-детектором на Go 1.22 и 1.23
- **Lint**: проверка форматирования и `go vet`
- **Build**: сборка бинарника и плагина

## Лицензия

MIT
