# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build
go build -o geo-dns .

# Run locally
./geo-dns

# Regenerate Swagger docs (after changing API annotations in api_handler.go)
swag init

# Docker
docker compose up --build
```

## Environment Variables

**Core:**

| Variable | Default | Description |
|---|---|---|
| `API_PORT` | `8080` | HTTP/HTTPS API port |
| `DNS_PORT` | `53` | DNS server port (UDP+TCP) |
| `API_SSL` | `false` | Set to `true` to enable HTTPS |
| `SSL_CERT_PATH` | — | Path to TLS certificate |
| `SSL_KEY_PATH` | — | Path to TLS private key |

**Local JWT auth** (используется когда `KEYCLOAK_ENABLED` не задан):

| Variable | Default | Description |
|---|---|---|
| `ADMIN_PASSWORD` | — | Пароль для `GET /login` (Basic Auth). Обязателен. |
| `JWT_SECRET` | `super-secret-key` | Секрет подписи JWT |

**Keycloak auth** (включается через `KEYCLOAK_ENABLED=true`):

| Variable | Description |
|---|---|
| `KEYCLOAK_ENABLED` | Установить `true` для включения Keycloak |
| `KEYCLOAK_URL` | Base URL Keycloak, напр. `https://auth.example.com` |
| `KEYCLOAK_REALM` | Имя realm, напр. `myrealm` |
| `KEYCLOAK_AUDIENCE` | Опционально: проверка поля `aud` токена (обычно client ID) |

При включённом Keycloak `/login` возвращает 410. Токены нужно получать напрямую у Keycloak. JWKS автоматически обновляются каждые 5 минут.

In Docker, ports are mapped as `1053:53` (DNS) and `8080:8080` (API).

## Architecture

The service runs two concurrent servers sharing a single `MemoryStorage` instance:

1. **DNS Server** (`dns_handler.go`) — handles UDP and TCP DNS queries via `github.com/miekg/dns`. For each query it: resolves the client's country via MaxMind GeoIP, looks up records with geo-tag priority (specific tag → `default`), and falls back to recursive resolution via `8.8.8.8` only for IPs in the whitelist.

2. **REST API** (`api_handler.go`) — chi router. Поддерживает два режима авторизации (см. ниже). Swagger UI доступен на `/swagger/`.

### Data Flow

```
DNS query → DNSHandler.ServeDNS
  → MemoryStorage.GetGeoTag(clientIP)   # MaxMind mmdb lookup
  → MemoryStorage.GetRecordsForQuery()  # geo-tag → "default" fallback
  → if whitelisted: recursive via 8.8.8.8
```

### Storage (`storage.go`)

`MemoryStorage` is the central state:
- `zones map[string][]Zone` — keyed by origin (e.g. `"example.com."`), each origin can have multiple zones with different `GeoTag` values.
- `allowedNet []string` — CIDR whitelist for recursive DNS (always includes `127.0.0.1/32`).
- `geoReader *geoip2.Reader` — MaxMind GeoIP2 reader, loaded from `data/geo-db.mmdb`.

State is persisted to disk immediately on every mutation:
- `data/zones.json` — all DNS zones
- `data/whitelist.json` — recursion ACL

### GeoIP (`geo_service.go`)

`GeoService.DownloadAndLoadDB(url)` downloads a `.mmdb.gz` file, decompresses it on-the-fly, saves to `data/geo-db.mmdb`, and hot-reloads it in `MemoryStorage` without restart.

### Key Models (`models.go`)

- `Zone` — top-level: `Origin` (e.g. `"example.com."`), `GeoTag` (ISO country code or `"default"`), `SOA`, `Records []ResourceRecord`.
- `ResourceRecord` — `Name`, `Type` (`"A"`, `"MX"`, `"TXT"`, etc.), `Value`, `TTL`.

## API Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/login` | No | Returns a JWT (24h expiry) |
| GET | `/zones` | JWT | List all zones |
| POST | `/zone` | JWT | Add/update a zone (body: `Zone` JSON) |
| POST | `/admin/allow?cidr=...` | JWT | Add CIDR to recursion whitelist |
| POST | `/geo/update?url=...` | JWT | Download & reload GeoIP DB |
| GET | `/swagger/*` | No | Swagger UI |

## Auth modes

`RegisterRoutes(r, tokenAuth, kc)` — если `kc != nil`, используется Keycloak middleware (`keycloak.go`); иначе — локальный `jwtauth`.

`KeycloakValidator` (`keycloak.go`) при старте загружает JWKS из `{KEYCLOAK_URL}/realms/{REALM}/protocol/openid-connect/certs`, проверяет подпись и `iss`. Если задан `KEYCLOAK_AUDIENCE` — также проверяет `aud`.

## GitFlow

Ветки:

| Ветка | Назначение |
|---|---|
| `main` | Только релизы. Прямые коммиты запрещены. |
| `develop` | Интеграционная ветка. Сюда мержатся все фичи. |
| `feature/*` | Новая функциональность. Ответвляется от `develop`, мержится в `develop`. |
| `release/*` | Подготовка релиза. От `develop` → мерж в `main` + `develop` + тег. |
| `hotfix/*` | Срочные правки прода. От `main` → мерж в `main` + `develop` + тег. |

**Цикл фичи:**
```bash
git checkout develop && git pull
git checkout -b feature/my-feature
# ... работа ...
gh pr create --base develop
# после merge PR:
git branch -d feature/my-feature
```

**Цикл релиза:**
```bash
git checkout -b release/vX.Y.Z develop
git push -u origin release/vX.Y.Z
# финальные правки если нужны, затем:
git checkout main && git merge --no-ff release/vX.Y.Z
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main && git push origin vX.Y.Z
git checkout develop && git merge --no-ff release/vX.Y.Z
git push origin develop
git push origin --delete release/vX.Y.Z
gh release create vX.Y.Z --title "vX.Y.Z" --target main
```

## Notes

- `docs/` is generated by `swag init` — do not edit manually.
- The `certs/` directory contains TLS cert/key for HTTPS mode; `data/` holds the GeoIP database and JSON persistence files. Both are bind-mounted in Docker.
