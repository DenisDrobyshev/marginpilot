# MarginPilot

**AI margin control plane — «Stripe для маржи на AI».**

B2B SaaS-компании массово встраивают AI-фичи и продают их по подписке, но не видят
свою unit-экономику: клиент на тарифе $99/мес может сжечь токенов на $300. MarginPilot
стоит между приложением и LLM-провайдерами, считает стоимость каждого запроса в разрезе
конкретного клиента и защищает валовую маржу — метерингом, бюджетами в реальном времени
и аналитикой P&L по каждому клиенту.

> Working codename. Модульный путь `github.com/marginpilot/*` меняется одной заменой,
> когда определишься с брендом.

## Почему это отдельная ниша

| Категория | Примеры | Чего не делают |
|-----------|---------|----------------|
| AI-gateway | LiteLLM, Portkey, Cloudflare | Прокси для разработчика; не мыслят клиентом/выручкой/маржой |
| Usage-биллинг | Lago, Metronome, Orb | Считают счета; не AI-native, не блокируют убыточный запрос в реальном времени |
| **MarginPilot** | — | Бизнес-outcome между ними: **знать и защищать маржу по каждому клиенту** |

Рынок подтверждён: по State of FinOps 2026 **98%** компаний управляют AI-расходами
(годом ранее — 63%), а рынок LLM-middleware растёт ~49.6% CAGR. Соседний биллинг
консолидируется гигантами (Stripe купил Metronome, Adyen — Orb), освобождая нишу
независимого AI-native metering + enforcement.

## Архитектура

Разбиение по **профилю нагрузки**, а не ради моды: data plane (низкая латентность,
stateless) → шина событий → control plane (строгая консистентность) → аналитика (ML).
Подробно — в [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
клиент → Gateway (Go) ⇄ LLM-провайдеры
              │  usage events
              ▼
        Kafka / Redpanda → Metering → ClickHouse / Redis
                                         │
              Rating → Billing → Stripe  │  control plane (Go)
                                         ▼
                    Margin Analytics (Python/ML) → Notifier
```

## Структура репозитория

```
marginpilot/
├── go.work                     # Go-воркспейс, связывает модули
├── docker-compose.yml          # весь стек одной командой
├── Makefile
├── proto/                      # ✅ contracts-модуль: .proto + сгенерированный gRPC (budget, identity)
├── shared/                     # общий Go-модуль: config, logger, httpx, events, spend (контракты)
├── services/
│   ├── gateway/                # ✅ data plane, гексагональная архитектура, покрыт тестами
│   │   ├── cmd/gateway/        # composition root (env-переключение адаптеров)
│   │   └── internal/
│   │       ├── domain/         # типы ядра (никаких HTTP/БД)
│   │       ├── port/           # выходные интерфейсы (LLMProvider, UsagePublisher, BudgetChecker)
│   │       ├── app/            # бизнес-логика (Proxy, CallerResolver) + unit-тесты
│   │       └── adapter/        # inbound (HTTP) + outbound (provider/publisher: stdout|kafka,
│   │                           #   budget: allow-all|grpc, resolver: header|identity-grpc)
│   ├── metering/               # ✅ Kafka-консьюмер → pricing → ClickHouse + Redis-спенд
│   ├── budget/                 # ✅ enforcement: Redis (спенд + rate-limit) + gRPC
│   ├── identity/               # ✅ резолв виртуальных ключей: Postgres + gRPC
│   ├── analytics/              # ✅ Python/FastAPI-скелет: маржа/ML
│   ├── rating/ billing/        # 🔜 control plane (спроектированы, README)
│   └── notifier/               # 🔜
└── docs/ARCHITECTURE.md
```

## Быстрый старт

### Локально (без Docker)

```bash
go run ./services/gateway/cmd/gateway      # :8080
```

В другом терминале — запрос через OpenAI-совместимый эндпоинт:

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-demo" \
  -H "X-Customer-Id: acme-inc" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"привет"}]}'
```

В логах gateway появится structured-событие `usage_event` с токенами и клиентом —
это то, что дальше поедет в metering.

### Весь стек в Docker (сквозной путь Фазы 2)

```bash
docker compose up --build -d
# infra: postgres, redis, clickhouse, redpanda
# сервисы: gateway, metering, budget, identity, analytics
# gateway проброшен на http://localhost:18080 (контейнерный порт 8080)
```

Теперь gateway работает с реальными адаптерами: ключ резолвится через **identity**
(gRPC→Postgres), usage-события летят в **Redpanda**, **metering** их тарифицирует,
пишет в **ClickHouse** и инкрементит спенд-счётчик в Redis, а **budget** (gRPC→Redis)
блокирует запросы сверх лимита.

```bash
# 1) валидный ключ sk-demo → резолвится в demo-tenant/acme-inc, запрос проходит
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:18080/v1/chat/completions \
  -H "Authorization: Bearer sk-demo" -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"привет"}]}'   # 200

# 2) неизвестный ключ → identity не находит → 401
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:18080/v1/chat/completions \
  -H "Authorization: Bearer nope" -d '{}'                                          # 401

# 3) метеринг зафиксировал стоимость в ClickHouse
docker compose exec clickhouse clickhouse-client -q \
  "SELECT count(), sum(cost_micros) FROM usage_events"

# 4) если поднять стек с крошечным лимитом — budget заблокирует перерасход:
#    BUDGET_DEFAULT_LIMIT_MICROS=1 docker compose up -d
#    после первого (оплаченного) запроса следующий вернёт 402 (budget exceeded).
```

### Тесты

```bash
make test
# или без make (в go-воркспейсе из корня нужно перечислить модули):
go test ./shared/... ./services/gateway/... ./services/metering/...
```

## Роадмап

- **Фаза 1 (ядро) — готова:** gateway-прокси, эмиссия usage-событий, каркас metering/analytics.
- **Фаза 2 (контроль) — готова:** реальный Redpanda-продюсер/консьюмер, тарификация + ClickHouse,
  бюджеты + rate-limit + enforcement (Redis, gRPC), identity (Postgres, gRPC). Осталось: notifier.
- **Фаза 3 (деньги):** каталог цен (rating), маржа/P&L по клиенту, экспорт usage в Stripe.
- **Фаза 4 (ML/enterprise):** прогноз спенда и anomaly-детекция (Python), семантический кэш, guardrails, SSO/SCIM.

## Технологии

Go 1.23 · Python 3.12 (FastAPI) · gRPC / protobuf · Kafka/Redpanda · ClickHouse · PostgreSQL ·
Redis · Docker Compose · гексагональная архитектура · event-driven / CQRS.

## Лицензия

MIT (open-core: ядро открыто, cloud/enterprise-фичи — платные). См. [LICENSE](LICENSE).
