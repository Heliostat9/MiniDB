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
- 🔐 Magic header и поддержка версий формата файла
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

## 🧱 Пример API (в коде)

```go
CreateTable("users", []string{"id INT", "name TEXT", "email TEXT"})
InsertRow("users", []string{"1", "Alice", "alice@example.com"})
SaveBinaryDB() // сохраняет в файл
```

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
