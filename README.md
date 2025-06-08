# 🧬 MiniDB

**MiniDB** — это минималистичная in-memory база данных, написанная на Go с бинарной сериализацией на диск.
Проект создавался с нуля — без использования сторонних движков, чтобы полностью контролировать формат хранения и логику.
Полная инструкция по запуску и командам находится в [docs/usage.md](docs/usage.md).

---

## 🚀 Возможности

- 📝 Создание таблиц с произвольными колонками
- 📥 Добавление строк в таблицу
- 🛠 Обновление существующих записей
- 💾 Сериализация таблиц в бинарный файл
- 📂 Загрузка таблиц при старте (persist между запусками)
- 📜 Журнал WAL для восстановления после сбоев
- 🔐 Magic header и поддержка версий формата файла
- Версия v3 хранит счётчики строк в 64 битах
- ⚙️ Написан чисто на Go (без зависимостей)
- 📊 Поддержка типов INT, FLOAT, BOOL и TEXT
- 📤 Экспорт таблиц в SQL-дамп

---

## 📦 Структура бинарного файла

Формат хранения:

| Magic Header | Version | Table 1 | Table 2 | … |

Каждая таблица включает:
- Имя таблицы
- Колонки
- Кол-во строк
- Строки с данными
- Журнал WAL хранится отдельно в `data.wal` и переигрывается при запуске

---

## 💻 Доступные CLI-команды

Создание таблицы:
```sql
CREATE TABLE <table_name> (, , …)
```

Добавление записи в таблицу:
```sql
INSERT INTO <table_name> VALUES (, , …)
```

Получение всех записей из таблицы:
```sql
SELECT * FROM <table_name>
```

Изменение записей:
```sql
UPDATE <table_name> SET <column>='<value>' WHERE <column>='<cond>'
```

Экспорт в SQL-дамп:
```sql
DUMP [filename]
```

Выход из БД:
```sql
EXIT
```
### 🧠 Пример

```sql
CREATE TABLE users (id, name, email)
CREATE TABLE metrics (score FLOAT, active BOOL)
INSERT INTO users VALUES (1, Alice, alice@example.com)
INSERT INTO metrics VALUES (3.14, true)
SELECT * FROM metrics
```

### 🧮 Примеры работы с типами данных

```sql
CREATE TABLE demo (
    n INT,
    rating FLOAT,
    active BOOL,
    note TEXT
)
INSERT INTO demo VALUES (42, 4.5, true, 'hello')
SELECT * FROM demo
```

## 🛠 Использование

```bash
go run main.go
```


> На текущем этапе взаимодействие происходит через встроенные вызовы (main.go), далее планируется добавить CLI / SQL-парсер.

## 🔄 CI/CD

Автоматические тесты запускаются через GitHub Actions при пуше и pull request в ветку `main`.

## 🛠 Инструменты разработки
- `golangci-lint` для статического анализа
- `pre-commit` хуки с `gofmt`, `govet` и `golangci-lint`
- `Makefile` с командами `make test`, `make lint`, `make coverage`
- `Dockerfile` для запуска в контейнере
- отчёты о покрытии тестами сохраняются как artifacts
- требует Go >=1.24

## 🧪 Запуск тестов

```bash
make test  # выполнить unit-тесты
make lint  # запустить линтер
```

## 🐳 Запуск в Docker

```bash
docker build -t minidb .
docker run -it --rm minidb
```

## 🧱 Пример API (в коде)

```go
CreateTable("users", []string{"id INT", "name TEXT", "email TEXT"})
InsertRow("users", []string{"1", "Alice", "alice@example.com"})
SaveBinaryDB() // сохраняет в файл
```

### Использование с `database/sql`

```go
import (
    "database/sql"
    _ "minisql/driver"
)

db, _ := sql.Open("minidb", "")
defer db.Close()
db.Exec("CREATE TABLE demo (id INT, name TEXT)")
db.Exec("INSERT INTO demo VALUES (1, 'Alice')")
row := db.QueryRow("SELECT * FROM demo")
```
Драйвер регистрируется автоматически при импорте. Установите его командой `go get <repo>/driver`, где `<repo>` — путь к репозиторию.

## 📁 Файл данных

По умолчанию сохраняется в файл data.mdb (бинарный формат).

## 📋 Планы на будущее
- SQL-парсинг (SELECT, INSERT, WHERE)
- CLI-интерфейс
- Тесты
- Поддержка разных типов данных

## 📄 Лицензия

MIT — используй, меняй, улучшай ✌️

## 🤝 Автор

> Артем Хитрин
Github: [Heliostat9](https://github.com/Heliostat9)
TG: [@heliostat](https://t.me/heliostat)
