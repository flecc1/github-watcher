# GitHub Watcher

A Telegram bot for monitoring GitHub repositories and tracking changes in their statistics over time.

Built with Go, the GitHub REST API, and PostgreSQL. The bot collects repository metrics on demand using a concurrent worker pool and stores every collection as a historical snapshot.

## Demo

![img_1.png](img/img_1.png)
![img_2.png](img/img_2.png)
![img_3.png](img/img_3.png)

## Features

- **Repository management** — add, list, and delete tracked repositories. Repositories are validated against the GitHub API before being saved, and names are matched case-insensitively.
- **Statistics collection** — stars, forks, open issues (excluding pull requests), open pull requests, README availability, number of commits in the last 7 days, and the date of the last commit.
- **Historical snapshots** — each `/collect` run stores a new snapshot in PostgreSQL; `/stats` shows the most recent one for every repository.
- **Concurrent processing** — repositories are processed by a fixed pool of 5 workers built on goroutines, channels, `select`, and `sync.WaitGroup`, with cancellation via `context`.
- **Partial failure handling** — an error in one repository does not stop collection for the others; the bot reports how many succeeded and how many failed.
- **Response caching** — successful GitHub API responses are cached in memory for 60 seconds (thread-safe cache built on `sync.RWMutex`).
- **Containerized deployment** — the bot and PostgreSQL run with Docker Compose; the schema is applied automatically on first start.
- **CI** — GitHub Actions checks formatting (`gofmt`), runs `go vet`, and builds the project.

## Tech Stack

| Technology                                                                           | Purpose                                           |
|--------------------------------------------------------------------------------------|---------------------------------------------------|
| Go 1.27                                                                              | Application logic and concurrent processing       |
| PostgreSQL 17                                                                        | Persistent storage for repositories and snapshots |
| [pgx](https://github.com/jackc/pgx) via `database/sql`                               | PostgreSQL driver                                 |
| [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)          | Telegram Bot API client                           |
| GitHub REST API                                                                      | Repository metadata and activity                  |
| `log/slog`                                                                           | Structured logging                                |
| Docker Compose                                                                       | Deployment of the bot and the database            |
| GitHub Actions                                                                       | Continuous integration                            |

## Architecture

```text
Telegram ──► bot.Handler ──► service.Service ──┬──► storage.Storage (interface) ──► postgres.Store ──► PostgreSQL
                                               │
                                               ├──► tracker.Tracker ──► github.Client ──► GitHub REST API
                                               │                              │
                                               │                              └──► cache.Cache (in-memory, TTL 60s)
                                               └──► github.Client
```

- `internal/bot` — parses Telegram commands, applies per-command timeouts, and formats replies.
- `internal/service` — business logic: repository validation, adding/deleting/listing repositories, the collection worker pool, and assembling statistics.
- `internal/tracker` — collects metrics for a single repository from the GitHub API and saves a snapshot.
- `internal/github` — GitHub REST API client with pagination and response caching.
- `internal/cache` — thread-safe in-memory cache with TTL.
- `internal/storage` — persistence contract (`Storage` interface) and domain errors.
- `internal/postgres` — `Storage` implementation on PostgreSQL.
- `internal/models` — repository and snapshot data structures.
- `internal/logger` — `slog` helpers.

### How `/collect` works

1. The service loads all tracked repositories from PostgreSQL.
2. A producer goroutine sends repositories into the `jobs` channel and closes it when done.
3. Five workers read from `jobs`. For each repository the tracker requests repository info, README, open pull requests, commits for the last 7 days, and the last commit, then saves a snapshot.
4. Workers send the result (success or error) into the `results` channel.
5. A separate goroutine waits for all workers (`sync.WaitGroup`) and then closes `results`.
6. The coordinator counts successes and failures; the bot edits its "collecting…" message with the summary.

The whole operation is limited to 30 seconds. On cancellation, workers stop through `select` on `ctx.Done()`, so no goroutines are left hanging.

## Project Structure

```text
.
├── .github/workflows/ci.yaml
├── Dockerfile
├── compose.yaml
├── .env.example
├── go.mod
├── go.sum
├── main.go
├── docs
│   ├── README.md
│   ├── PLAN.md
│   └── img/
├── internal
│   ├── bot
│   │   └── handlers.go
│   ├── cache
│   │   └── cache.go
│   ├── github
│   │   ├── client.go
│   │   ├── commits.go
│   │   ├── openPullRequests.go
│   │   └── repositories.go
│   ├── logger
│   │   └── logger.go
│   ├── models
│   │   ├── repository.go
│   │   └── repositorySnapshot.go
│   ├── postgres
│   │   ├── postgres.go
│   │   └── repositories.go
│   ├── service
│   │   ├── service.go
│   │   └── service_test.go
│   ├── storage
│   │   └── storage.go
│   └── tracker
│       └── tracker.go
└── migrations
    └── scheme.sql
```

## Getting Started

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)

### 1. Clone the repository

```bash
git clone https://github.com/flecc1/github-watcher.git
cd github-watcher
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

| Variable             | Description                                                    |
|----------------------|----------------------------------------------------------------|
| `TELEGRAM_BOT_TOKEN` | Bot token from @BotFather                                      |
| `POSTGRES_PASSWORD`  | Password for the `watcher` database user (used by Compose)     |
| `DATABASE_URL`       | PostgreSQL connection string used by the bot                   |

```dotenv
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
POSTGRES_PASSWORD=your_database_password
DATABASE_URL=postgres://watcher:your_database_password@db:5432/github_watcher
```

The password in `DATABASE_URL` must match `POSTGRES_PASSWORD`.

### 3. Run with Docker Compose

```bash
docker compose up --build -d
```

Docker Compose starts PostgreSQL, applies `migrations/scheme.sql` on the first start of an empty database, and starts the bot once PostgreSQL is healthy.

Open your bot in Telegram and send `/start`.

### Running locally without Docker

1. Start PostgreSQL and apply the schema:

   ```bash
   psql "$DATABASE_URL" -f migrations/scheme.sql
   ```

2. Set `DATABASE_URL` in `.env` to point at your local database (e.g. `postgres://watcher:password@localhost:5432/github_watcher`). The `.env` file is loaded automatically on startup.
3. Run the bot:

   ```bash
   go run .
   ```

## Bot Commands

| Command                 | Description                                         |
|-------------------------|-----------------------------------------------------|
| `/start`                | Greeting and quick start                            |
| `/help`                 | Show available commands                             |
| `/add owner/repo`       | Add a repository to track                           |
| `/delete owner/repo`    | Stop tracking a repository (its snapshots are removed too) |
| `/list`                 | List tracked repositories                           |
| `/collect`              | Collect fresh statistics and save a snapshot        |
| `/stats`                | Show the latest saved statistics for each repository |

## Database

- `repositories` — tracked repositories (`owner`, `name`, `full_name`, `added_at`). A unique index on `LOWER(full_name)` prevents duplicates regardless of case.
- `repository_snapshots` — collected metrics: stars, forks, open issues, open pull requests, README availability, commits in the last 7 days, last commit date, and collection time.

Snapshots reference repositories with `ON DELETE CASCADE`, so deleting a repository removes its history. An index on `(repository_id, collected_at DESC)` speeds up fetching the latest snapshots. Data is kept in a Docker volume between restarts.

> The schema is applied only when PostgreSQL initializes an **empty** volume. After changing `migrations/scheme.sql`, either apply the changes manually or recreate the volume with `docker compose down -v` (this deletes all data).

## Testing

```bash
go test ./...
go test -race ./...
```

Unit tests cover the service layer (repository name parsing, `Delete`, `List`) using a fake implementation of the `Storage` interface.

## Limitations

- GitHub API requests are unauthenticated, so the limit is 60 requests per hour. Each repository takes at least 5 requests per collection.
- Statistics are collected only on demand via `/collect`; there is no scheduled collection.
- All chats share one list of tracked repositories.
