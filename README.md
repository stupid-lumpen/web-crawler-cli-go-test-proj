# Crawler CLI

*CLI-приложение на Go для асинхронного обхода сайтов и сохранения дерева ссылок в JSON.
*Требования

*Go 1.22 или выше

*Make (опционально)

## Сборка

### Сборка бинарного файла в папку bin/

    make build

### Или через Go

    go build -o bin/crawler ./cmd/crawler

## Запуск

### Пример запуска с флагами

    ./bin/crawler

    --urls <https://google.com,https://example.com>

    --depth 3

    --timeout 2m

    --request-timeout 10s

    --output result.json

    --log crawler.log

## Флаги

    --urls — список стартовых URL через запятую.

    --depth — глубина обхода.

    --timeout — общий таймаут работы приложения.

    --request-timeout — таймаут на один HTTP-запрос.

    --output — путь к итоговому JSON-файлу.

    --log — путь к лог-файлу для ошибок и HTTP-статусов.

## Запуск напрямую из исходников

    make run

## Тесты

### Запуск unit-тестов

    make test

### Проверка покрытия тестами

    make test-coverage

### Очистка сгенерированных файлов

    make clean
