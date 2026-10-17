[![Go Reference](https://img.shields.io/badge/godoc-reference-blue)](https://pkg.go.dev/github.com/goloop/observe) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/observe/blob/v1/LICENSE) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# observe

`observe` дає кожному Go-сервісу однакові операційні endpoint-и - liveness,
readiness і build-info - без копіпасти хендлерів. Нуль залежностей, лише
стандартна бібліотека.

Це свідомо **не** платформа метрик/трейсингу: без Prometheus-клієнта, без
OpenTelemetry SDK. Лишається сумісним із ними, не залежачи від них.

## Встановлення

```bash
go get github.com/goloop/observe
```

## Швидкий старт

```go
obs := observe.New(
	observe.WithBuildInfo(observe.BuildInfo{Service: "api", Version: "v1.2.3"}),
	observe.WithErrorRedaction(true), // ховати деталі у prod
)

obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })
obs.Check("redis", func(ctx context.Context) error { return rdb.Ping(ctx) })

mux.Handle("/healthz", obs.HealthHandler()) // 200, поки процес живий
mux.Handle("/readyz", obs.ReadyHandler())   // 503, якщо залежність down
mux.Handle("/buildz", obs.BuildHandler())
```

## Health проти readiness

- **Health** (`/healthz`) - liveness рівня процесу: «чи процес живий». Не
  запускає перевірки залежностей і завжди `200`.
- **Readiness** (`/readyz`) виконує зареєстровані перевірки паралельно (кожна в
  межах таймауту) і повертає `503`, коли якась залежність down, щоб балансувальник
  припинив слати трафік.

Форма відповіді:

```json
{
  "status": "error",
  "service": "api",
  "version": "v1.2.3",
  "checks": {
    "postgres": {"status": "ok", "latency_ms": 3},
    "redis": {"status": "error", "error": "dial timeout"}
  }
}
```

`WithErrorRedaction(true)` лишає статус помилки, але прибирає текст `error`.

## Незв'язаність

`observe` не імпортує lifecycle-ядро. Щоб віддати стан життєвого циклу
застосунку, зареєструйте його як звичайну перевірку над знімком статусу:

```go
obs.Check("app", func(context.Context) error {
	if !a.Status().Healthy() {
		return errors.New("компонент впав")
	}
	return nil
})
```

## Документація

- Англійський довідник: [DOC.md](DOC.md)
- Український довідник: [DOC.UK.md](DOC.UK.md)

## Ліцензія

MIT - див. [LICENSE](LICENSE).

## Реліз v1.0.0

Поточний API оголошено стабільним без змін алгоритмів. Реліз v1 зберігає поточний import path і публічний API. Майбутні несумісні зміни потребуватимуть нової головної версії.
