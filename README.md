# geo-dns

DNS-сервер с геолокацией клиентов и REST API для управления зонами. Возвращает разные DNS-записи в зависимости от страны запрашивающего IP.

## Возможности

- DNS-сервер (UDP + TCP) с поддержкой A, AAAA, CNAME, MX, NS, TXT записей
- Геолокация клиентов через [MaxMind GeoIP2](https://www.maxmind.com/) (`.mmdb`)
- Geo-теги на зонах: `RU`, `US`, `DE` и т.д.; запись `default` — для всех остальных
- AXFR (zone transfer) — только для IP из whitelist
- REST API для управления зонами (chi + JWT)
- Два режима авторизации API: локальный JWT или Keycloak (OIDC)
- Горячая перезагрузка GeoIP базы без перезапуска
- Хранение зон в памяти с персистентностью в JSON
- Docker / docker-compose

## Быстрый старт

```bash
cp .env.example .env
# Заполнить .env

docker compose up --build
```

DNS будет доступен на порту `1053`, API — на `8080`.

## Конфигурация

Все параметры задаются через `.env` (см. [.env.example](.env.example)).

**Основные:**

| Переменная | По умолчанию | Описание |
|---|---|---|
| `API_PORT` | `8080` | Порт REST API |
| `DNS_PORT` | `53` | Порт DNS-сервера (UDP + TCP) |
| `API_SSL` | `false` | Включить HTTPS |
| `SSL_CERT_PATH` | — | Путь к TLS-сертификату |
| `SSL_KEY_PATH` | — | Путь к TLS-ключу |

**Локальная JWT авторизация** (по умолчанию):

| Переменная | Описание |
|---|---|
| `ADMIN_PASSWORD` | Пароль для `GET /login` (Basic Auth) |
| `JWT_SECRET` | Секрет подписи JWT |

**Keycloak авторизация** (опционально):

| Переменная | Описание |
|---|---|
| `KEYCLOAK_ENABLED` | Установить `true` для включения |
| `KEYCLOAK_URL` | Base URL Keycloak, напр. `https://auth.example.com` |
| `KEYCLOAK_REALM` | Имя realm |
| `KEYCLOAK_AUDIENCE` | Опционально: проверка поля `aud` токена |

## API

Swagger UI доступен по адресу `http://localhost:8080/swagger/`.

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/login` | Получить JWT (Basic Auth: `admin` / `ADMIN_PASSWORD`) |
| `GET` | `/zones` | Список всех зон |
| `POST` | `/zone` | Добавить / обновить зону |
| `POST` | `/admin/allow?cidr=` | Добавить CIDR в whitelist рекурсии |
| `POST` | `/geo/update?url=` | Скачать и загрузить новую GeoIP базу |

Все эндпоинты кроме `/login` и `/swagger/` требуют заголовок `Authorization: Bearer <token>`.

### Пример: добавление зоны

```bash
TOKEN=$(curl -s -u admin:changeme http://localhost:8080/login)

curl -s -X POST http://localhost:8080/zone \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "example.com.",
    "geo_tag": "RU",
    "soa": {
      "ns": "ns1.example.com.",
      "mbox": "admin.example.com.",
      "serial": 2024010101,
      "refresh": 86400,
      "retry": 7200,
      "expire": 3600000,
      "minttl": 300
    },
    "records": [
      {"name": "example.com.", "type": "A", "value": "1.2.3.4", "ttl": 300}
    ]
  }'
```

Та же зона с `"geo_tag": "default"` будет отдаваться всем остальным клиентам.

### Формат MX-записи

Поле `value` для MX: `"<priority> <hostname>"`, например:

```json
{"name": "example.com.", "type": "MX", "value": "10 mail.example.com.", "ttl": 300}
```

### Обновление GeoIP базы

```bash
curl -s -X POST \
  "http://localhost:8080/geo/update?url=https://example.com/dbip-country-lite.mmdb.gz" \
  -H "Authorization: Bearer $TOKEN"
```

Поддерживаются `.mmdb.gz` файлы. База обновляется в памяти без перезапуска.

## Whitelist

Whitelist управляет двумя функциями: рекурсивными DNS-запросами и AXFR (zone transfer). По умолчанию разрешён только `127.0.0.1/32`.

Добавить сеть:

```bash
curl -s -X POST "http://localhost:8080/admin/allow?cidr=10.0.0.0/8" \
  -H "Authorization: Bearer $TOKEN"
```

**Рекурсия** — для запросов, которых нет в локальных зонах, сервер обращается к `8.8.8.8` только если IP клиента в whitelist. Остальным возвращается `REFUSED`.

**AXFR** — полный дамп зоны доступен только из whitelist:

```bash
dig axfr example.com. @localhost -p 1053
```

Ответ содержит: SOA → все записи зоны → SOA. Зона выбирается с geo-приоритетом (тег клиента → `default`).

## Локальная разработка

```bash
go build -o geo-dns .
ADMIN_PASSWORD=secret ./geo-dns

# После изменения аннотаций Swagger в api_handler.go:
swag init
```
