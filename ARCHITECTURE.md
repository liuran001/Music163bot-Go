# Architecture

## Overview

This project is structured around a layered architecture to keep Telegram transport, NetEase API calls, and persistence concerns separated.

## Directory Layout

```
main.go                  # Application entry point
internal/
  app/                   # Dependency injection and lifecycle
  config/                # Configuration (Viper + INI)
  db/                    # SQLite/GORM repositories
  logger/                # slog logging
  netease/               # NetEase API client with retries + circuit breaker
  telegram/              # go-telegram/bot integration
  telegram/handler/      # Command handlers
  worker/                # Bounded concurrency pool
  updater/               # Dynamic update abstraction (stub)
tests/
  integration/           # Integration tests
```

## Core Flow

1. `main.go` creates `app.App` with build metadata and config path.
2. `app.Start` wires dependencies and registers handlers.
3. Telegram updates are routed to handlers under `internal/telegram/handler`.
4. Handlers call `netease.Client` and `db.Repository` for data and persistence.

## Key Modules

- `internal/telegram/handler/music.go`: Core download/send flow
- `internal/netease/client.go`: API wrapper with retry + circuit breaker
- `internal/db/repository.go`: Cache operations and SQLite PRAGMAs

## Notes

- Legacy `bot/` package已移除，代码全部迁移到 `internal/`.
- Dynamic update support is abstracted in `internal/updater/` for future integration.
