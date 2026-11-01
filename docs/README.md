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
- **Clear error messages** — invalid input, already tracked, and not tracked repositories get dedicated replies; only unexpected errors are logged.
- **GitHub authentication** — an optional personal access token raises the GitHub API limit from 60 to 5,000 requests per hour.
- **Response caching** — successful GitHub API responses are cached in memory for 60 seconds (thread-safe cache built on `sync.RWMutex`).
- **Database migrations** — the schema is versioned with [golang-migrate](https://github.com/golang-migrate/migrate); every change has `up` and `down` scripts.
- **Graceful shutdown** — on `SIGINT`/`SIGTERM` the bot stops receiving updates and finishes the update it is processing before exiting.
- **Containerized deployment** — PostgreSQL, migrations, and the bot run with Docker Compose.
- **CI** — GitHub Actions checks formatting (`gofmt`), runs `go vet`, builds the project, and runs tests with the race detector.

## Tech Stack

| Technology                                                                  | Purpose                                           |
|-----------------------------------------------------------------------------|---------------------------------------------------|
| Go 1.27                                                                     | Application logic and concurrent processing       |
| PostgreSQL 17                                                               | Persistent storage for repositories and snapshots |
| [pgx](https://github.com/jackc/pgx) via `database/sql`                      | PostgreSQL driver                                 |
| [golang-migrate](https://github.com/golang-migrate/migrate)                 | Database schema migrations                        |
| [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) | Telegram Bot API client                           |
| GitHub REST API                                                             | Repository metadata and activity                  |
| `log/slog`                                                                  | Structured logging                                |
| Docker Compose                                                              | Running the database, migrations, and the bot     |
| GitHub Actions                                                              | Continuous integration                            |

## Architecture

```text
Telegram ──► bot.Handler ──► service.Service ──┬──► storage.Storage (interface) ──► postgres.Store ──► PostgreSQL
                                               │
                                               ├──► tracker.Tracker ──► github.Client ──► GitHub REST API
                                               │                              │
                                               │                              └──► cache.Cache (in-memory, TTL 60s)
                                               └──► github.Client
```

- `internal/bot` — parses Telegram commands, applies per-command timeouts, maps errors to user-facing replies, and formats responses.
- `internal/service` — business logic: repository name validation, adding/deleting/listing repositories, the collection worker pool, and assembling statistics.
- `internal/tracker` — collects metrics for a single repository from the GitHub API and saves a snapshot.
- `internal/github` — GitHub REST API client with optional token authentication, pagination, and response caching.
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
├── db
│   └── migrations
│       ├── 000001_create_repositories.{up,down}.sql
│       ├── 000002_create_repository_snapshots.{up,down}.sql
│       └── 000003_timestamptz.{up,down}.sql
├── docs
│   ├── README.md
│   ├── PLAN.md
│   └── img/
└── internal
    ├── bot
    │   └── handlers.go
    ├── cache
    │   └── cache.go
    ├── github
    │   ├── client.go
    │   ├── commits.go
    │   ├── openPullRequests.go
    │   └── repositories.go
    ├── logger
    │   └── logger.go
    ├── models
    │   ├── repository.go
    │   └── repositorySnapshot.go
    ├── postgres
    │   ├── postgres.go
    │   └── repositories.go
    ├── service
    │   ├── service.go
    │   └── service_test.go
    ├── storage
    │   └── storage.go
    └── tracker
        └── tracker.go
```

## Getting Started

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- Optional: a GitHub [personal access token](https://github.com/settings/tokens) (no scopes are needed for public repositories)

### 1. Clone the repository

```bash
git clone https://github.com/flecc1/github-watcher.git
cd github-watcher
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

| Variable             | Required | Description                                                         |
|----------------------|----------|---------------------------------------------------------------------|
| `TELEGRAM_BOT_TOKEN` | yes      | Bot token from @BotFather                                           |
| `DATABASE_URL`       | yes      | PostgreSQL connection string used by the bot and by migrations      |
| `POSTGRES_PASSWORD`  | yes      | Password for the `watcher` database user (used by Docker Compose)   |
| `GITHUB_TOKEN`       | no       | GitHub token; without it the GitHub API allows 60 requests per hour |

```dotenv
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
GITHUB_TOKEN=your_github_token
POSTGRES_PASSWORD=your_database_password
DATABASE_URL=postgres://watcher:your_database_password@db:5432/github_watcher?sslmode=disable
```

The password in `DATABASE_URL` must match `POSTGRES_PASSWORD`. Keep `?sslmode=disable`: the PostgreSQL container does not use SSL, and golang-migrate requires it by default.

### 3. Run with Docker Compose

```bash
docker compose up --build -d
```

Compose starts the services in order:

1. `db` — PostgreSQL; the next step waits until its health check passes.
2. `migrate` — applies new migrations from `db/migrations` and exits. If nothing changed, it reports `no change`.
3. `bot` — starts only after migrations have completed successfully.

Open your bot in Telegram and send `/start`.

### Running locally without Docker

1. Install the [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) (for example, `brew install golang-migrate`).
2. Start PostgreSQL and set `DATABASE_URL` in `.env` to point at it, for example `postgres://watcher:password@localhost:5432/github_watcher?sslmode=disable`. The `.env` file is loaded automatically on startup.
3. Apply migrations:

   ```bash
   migrate -path db/migrations -database "$DATABASE_URL" up
   ```

4. Run the bot:

   ```bash
   go run .
   ```

## Bot Commands

| Command              | Description                                                |
|----------------------|------------------------------------------------------------|
| `/start`             | Greeting and quick start                                   |
| `/help`              | Show available commands                                    |
| `/add owner/repo`    | Add a repository to track                                  |
| `/delete owner/repo` | Stop tracking a repository (its snapshots are removed too) |
| `/list`              | List tracked repositories                                  |
| `/collect`           | Collect fresh statistics and save a snapshot               |
| `/stats`             | Show the latest saved statistics for each repository       |

## Database

- `repositories` — tracked repositories (`owner`, `name`, `full_name`, `added_at`). A unique index on `LOWER(full_name)` prevents duplicates regardless of case.
- `repository_snapshots` — collected metrics: stars, forks, open issues, open pull requests, README availability, commits in the last 7 days, last commit date, and collection time.

Snapshots reference repositories with `ON DELETE CASCADE`, so deleting a repository removes its history. An index on `(repository_id, collected_at DESC)` speeds up fetching the latest snapshots. All timestamps are stored as `TIMESTAMPTZ`. Data is kept in a Docker volume between restarts.

### Migrations

| Version | Migration                       | Change                                                    |
|---------|---------------------------------|-----------------------------------------------------------|
| 1       | `create_repositories`           | `repositories` table and case-insensitive unique index    |
| 2       | `create_repository_snapshots`   | `repository_snapshots` table and index                    |
| 3       | `timestamptz`                   | All timestamp columns converted to `TIMESTAMPTZ` (as UTC) |

The applied version is stored in the `schema_migrations` table. To change the schema, add a new pair of files instead of editing applied ones:

```bash
migrate create -ext sql -dir db/migrations -seq <description>
```

Roll back the last migration:

```bash
migrate -path db/migrations -database "$DATABASE_URL" down 1
```

## Testing

```bash
go test ./...
go test -race ./...
```

Unit tests cover the service layer (repository name parsing, `Delete`, `List`) using a fake implementation of the `Storage` interface. CI runs the tests with the race detector on every push and pull request to `main`.

## Limitations

- Statistics are collected only on demand via `/collect`; there is no scheduled collection.
- All chats share one list of tracked repositories.
- Each repository takes at least 5 GitHub API requests per collection, so without `GITHUB_TOKEN` only a few repositories can be collected per hour.
