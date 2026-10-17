# observe - довідник

`observe` володіє моделлю операційного статусу сервісу: liveness, readiness і
build-info. Повний український довідник; англійською - [DOC.md](DOC.md).

## Зміст

- [Модель](#модель)
- [Створення й опції](#створення-й-опції)
- [Реєстрація перевірок](#реєстрація-перевірок)
- [Health, readiness, build](#health-readiness-build)
- [HTTP-хендлери](#http-хендлери)
- [Формат відповіді](#формат-відповіді)
- [Незв'язаність із lifecycle-ядром](#незвязаність-із-lifecycle-ядром)
- [Конкурентність](#конкурентність)
- [Межі](#межі)

## Модель

`Registry` тримає іменовані readiness-перевірки й build-метадані. Розрізняє два
питання:

- **Liveness** (`Health`): чи процес живий? Завжди ok; перевірок не запускає.
- **Readiness** (`Ready`): чи залежності придатні? Виконує всі перевірки і падає,
  якщо якась впала.

Цей поділ важливий для оркестрації: залежність, що впала, має вивести под із
ротації балансувальника (readiness), не вбиваючи процес (liveness).

## Створення й опції

```go
obs := observe.New(opts...)
```

| Опція | Ефект | Дефолт |
|-------|-------|--------|
| `WithService(name)` | ім'я сервісу у відповідях | "" |
| `WithBuildInfo(BuildInfo)` | build-метадані; задає service, якщо порожній | zero |
| `WithCheckTimeout(d)` | дедлайн однієї перевірки в Ready | 2s |
| `WithErrorRedaction(bool)` | ховати текст помилки перевірки | false |

`BuildInfo` - це `{Service, Version, Commit, BuiltAt}`; заповнюйте через
`-ldflags` або `runtime/debug.ReadBuildInfo`.

## Реєстрація перевірок

```go
obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })
```

`Check` - це `func(context.Context) error`: nil означає healthy. Повторна
реєстрація тим самим іменем замінює. nil-функція ігнорується. Перевірки мають
бути швидкими й поважати дедлайн контексту.

## Health, readiness, build

```go
obs.Health(ctx) Status // liveness, завжди ok, без перевірок
obs.Ready(ctx)  Status // виконує всі перевірки паралельно, агрегує
obs.Build()     BuildInfo
```

`Ready` виконує кожну перевірку в окремій горутині, кожну в межах check-таймауту
(контекст запиту теж діє). Межа реальна: перевірка, що ігнорує свій контекст і
зависає, все одно дає помилку таймауту, не блокуючи відповідь (її горутина
протікає, доки перевірка колись не повернеться, але readiness відповідає).
Перевірка, що панікує, відновлюється й повідомляється як помилка, а не валить
процес. Загальний статус `error`, якщо якась перевірка впала, інакше `ok`.
Порожній реєстр - ready.

## HTTP-хендлери

```go
mux.Handle("/healthz", obs.HealthHandler()) // завжди 200
mux.Handle("/readyz", obs.ReadyHandler())   // 200 якщо все ok, інакше 503
mux.Handle("/buildz", obs.BuildHandler())   // 200, build-info
```

Кожен пише індентований JSON із `Content-Type: application/json`.

## Формат відповіді

```json
{
  "status": "ok",
  "service": "api",
  "version": "v1.2.3",
  "checks": {
    "postgres": {"status": "ok", "latency_ms": 3},
    "redis": {"status": "error", "latency_ms": 200}
  }
}
```

Кожна перевірка повідомляє `latency_ms`. `Health`/`Build` не містять `checks`.
Редакція **увімкнена за замовчуванням**: перевірка, що впала, дає
`{"status":"error","latency_ms":N}` без поля `error`, тож внутрішні деталі
(хости, DSN, креденшали) не витікають з health-ендпоінта. Передайте
`WithErrorRedaction(false)`, щоб додати поле `"error"` для локального
дебагу.

## Незв'язаність із lifecycle-ядром

`observe` не імпортує lifecycle-ядро і навпаки. Містком служить перевірка над
знімком статусу у вашому коді обв'язки:

```go
obs.Check("app", func(context.Context) error {
	if !a.Status().Healthy() {
		return errors.New("компонент впав")
	}
	return nil
})
```

Так обидва пакети лишаються незалежними й без залежностей.

## Конкурентність

`Registry` безпечний для конкурентного вжитку: `Check` бере write-lock;
`Health`, `Ready` і `Build` беруть read-lock і виконують перевірки на копії-
знімку, тож реєструвати й обслуговувати одночасно безпечно.

## Межі

`observe` робить: health/readiness/build endpoint-и, перевірки залежностей із
таймаутом і редакцією. Майбутня опційна хвиля може додати крихітний
request-metrics middleware з обмеженими лейблами.

`observe` не робить: OpenTelemetry, distributed tracing, alerting, дашборди чи
log-агрегацію. Лишається сумісним із ними, не залежачи від них.

## Реліз v1.0.0

Поточний API оголошено стабільним без змін алгоритмів. Реліз v1 зберігає поточний import path і публічний API. Майбутні несумісні зміни потребуватимуть нової головної версії.
