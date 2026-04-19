# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL из `migrations/*.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Периодичность задач

Поддерживаются правила из ТЗ:

| Тип (`recurrence.kind`) | Смысл |
| --- | --- |
| `daily_interval` | Каждый **n-й день**: `every_n_days` + обязательный `anchor_date` (YYYY-MM-DD, UTC). |
| `monthly_day` | Раз в месяц в заданное **число** `day_of_month` **1–30** (если в месяце такого дня нет — месяц пропускается). |
| `specific_dates` | Только перечисленные даты в `dates`. |
| `day_parity` | Только **чётные** или только **нечётные** числа месяца (`parity`: `even` / `odd`). |

Модель данных:

- Строка-**шаблон** (`is_template=true`) хранит название, описание и JSON правила `recurrence`.
- Для каждого подходящего календарного дня создаются **экземпляры** (`is_template=false`) с полем `occurrence_date`, связью `template_id` и общим `series_id`.
- При создании задачи с `recurrence` сервер **сразу генерирует экземпляры** на горизонт **`materialize_horizon_days`** (если не указан — **30** полных UTC-дней от сегодняшней даты).
- Дополнительно можно догонять экземпляры: **`POST /api/v1/tasks/{id}/materialize`** с телом `{"from":"YYYY-MM-DD","to":"YYYY-MM-DD"}` ( `id` — шаблон). Повторный вызов с теми же датами **идемпотентен** (дубликаты по паре `series_id` + `occurrence_date` не создаются).

Список задач по умолчанию **скрывает шаблоны**. Чтобы увидеть их: `GET /api/v1/tasks?include_templates=true`. Фильтр по календарю: `occurrence_from`, `occurrence_to` (UTC даты); **разовые** задачи без `occurrence_date` по-прежнему попадают в выдачу.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks` — можно передать `recurrence` (+ опционально `materialize_horizon_days`).
- `GET /api/v1/tasks` — опционально `include_templates`, `occurrence_from`, `occurrence_to`.
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}` — для шаблона можно обновить `recurrence`.
- `DELETE /api/v1/tasks/{id}`
- `POST /api/v1/tasks/{id}/materialize` — догenerate экземпляры по шаблону за интервал `from`/`to`.
