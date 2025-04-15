# 🧬 MiniDB

**MiniDB** — это минималистичная in-memory база данных, написанная на Go с бинарной сериализацией на диск.  
Проект создавался с нуля — без использования сторонних движков, чтобы полностью контролировать формат хранения и логику.

---

## 🚀 Возможности

- 📝 Создание таблиц с произвольными колонками
- 📥 Добавление строк в таблицу
- 💾 Сериализация таблиц в бинарный файл
- 📂 Загрузка таблиц при старте (persist между запусками)
- 🔐 Magic header и поддержка версий формата файла
- ⚙️ Написан чисто на Go (без зависимостей)

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

Выход из БД:
```sql
EXIT
```
### 🧠 Пример

```sql
CREATE TABLE users (id, name, email)
INSERT INTO users VALUES (1, Alice, alice@example.com)
INSERT INTO users VALUES (2, Bob, bob@example.com)
SELECT * FROM users
```

## 🛠 Использование

```bash
go run main.go
```


> На текущем этапе взаимодействие происходит через встроенные вызовы (main.go), далее планируется добавить CLI / SQL-парсер.

## 🧱 Пример API (в коде)

```go
CreateTable("users", []string{"id", "name", "email"})
InsertRow("users", []string{"1", "Alice", "alice@example.com"})
SaveBinaryDB() // сохраняет в файл
```

## 📁 Файл данных

По умолчанию сохраняется в файл data.mdb (бинарный формат).

## 📋 Планы на будущее
- SQL-парсинг (SELECT, INSERT, WHERE)
- CLI-интерфейс
- Тесты
- Dump в .sql
- Поддержка разных типов данных

## 📄 Лицензия

MIT — используй, меняй, улучшай ✌️

## 🤝 Автор

> Артем Хитрин
Github: [Heliostat9](https://github.com/Heliostat9)
TG: [@heliostat](https://t.me/heliostat)