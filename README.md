# GitHub Watcher 🤖

A Telegram bot written in Go that tracks GitHub repositories and collects their statistics, storing historical snapshots in PostgreSQL.

The project is being developed as a practical way to learn Go backend development, work with external APIs, and explore concurrency.

## Features

- **Repository tracking** — add GitHub repositories using the `owner/repo` format.
- **Repository validation** — check repository information through the GitHub API before adding it.
- **PostgreSQL storage** — persist tracked repositories and their statistics.
- **Statistics collection** — collect repository metrics and save snapshots.
- **Historical snapshots** — store multiple collections for each repository.
- **Repository management** — list tracked repositories and remove them.
- **Telegram interface** — interact with the bot through commands.
- **Context-aware operations** — pass `context.Context` through service and storage operations.
- **Structured logging** — use Go's `log/slog` for application logs.
- **Error handling** — handle collection failures without preventing the processing of other repositories.

## Collected Statistics

For each repository, the bot collects:

- ⭐ Stars
- 🍴 Forks
- 🐛 Open issues
- 🔀 Open pull requests
- 💻 Commits during the last 7 days
- 📄 README availability
- 🕒 Last commit timestamp

Each successful collection is stored as a separate snapshot in PostgreSQL.

## Tech Stack

- **Go**
- **PostgreSQL**
- **GitHub API**
- **Telegram Bot API**
- `database/sql`
- `pgx` PostgreSQL driver
- `go-telegram-bot-api/v5`
- `godotenv`
- `log/slog`

## Telegram Commands

| Command | Description |
|---|---|
| `/start` | Start interacting with the bot |
| `/help` | Show available commands |
| `/add owner/repo` | Add a repository to track |
| `/list` | List tracked repositories |
| `/collect` | Collect and save fresh statistics |
| `/stats` | Display the latest saved statistics |

## Project Structure

```text
github-watcher/
├── main.go
├── models/
│   └── ...
├── internal/
│   ├── bot/
│   │   └── ...
│   ├── github/
│   │   └── ...
│   ├── postgres/
│   │   └── ...
│   ├── repository/
│   │   └── ...
│   ├── storage/
│   │   └── ...
│   └── tracker/
│       └── ...
├── .env
├── .gitignore
├── go.mod
└── README.md
```

- `models` — domain models for repositories and statistics snapshots.
- `internal/bot` — Telegram command handling and message formatting.
- `internal/github` — interaction with the GitHub API.
- `internal/postgres` — PostgreSQL connection and persistence implementation.
- `internal/repository` — repository management and validation.
- `internal/storage` — storage interface and storage-related contracts.
- `internal/tracker` — collection of repository statistics.
- `main.go` — initialization and dependency wiring.

## Getting Started

### Prerequisites

- Go installed
- PostgreSQL running
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- GitHub API access

### 1. Clone the repository

```bash
git clone https://github.com/flecc1/github-watcher.git
cd github-watcher
```

### 2. Configure environment variables

Create a `.env` file in the project root:

```env
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
DATABASE_URL=your_postgres_connection_string
```

Use the actual environment variable names expected by the application. Do not commit `.env` or expose your tokens and database credentials.

### 3. Prepare the database

Make sure PostgreSQL is running and the required database schema and tables have been created.

The exact schema and migration setup depend on the current database initialization implementation.

### 4. Install dependencies

```bash
go mod download
```

### 5. Run the bot

```bash
go run .
```

After startup, open your bot in Telegram and send `/start`.

## Example Usage

Add a repository:

```text
/add golang/go
```

List tracked repositories:

```text
/list
```

Collect fresh statistics:

```text
/collect
```

Display the latest saved statistics:

```text
/stats
```

The `/collect` command reports successful and failed collections. If a collection fails, a previously saved snapshot may still be available through `/stats`.

## Development Status

The project is under active development.

### Implemented

- [x] Telegram bot initialization
- [x] PostgreSQL integration
- [x] Repository and snapshot models
- [x] Storage interface and PostgreSQL implementation
- [x] Repository validation and management
- [x] GitHub API integration
- [x] Repository statistics collection
- [x] Snapshot persistence
- [x] Basic Telegram commands
- [x] Error handling and structured logging
- [x] Display of the latest saved statistics

### Planned

- [ ] **Concurrent collection** — collect statistics for multiple repositories using goroutines.
- [ ] **Worker pool** — limit the number of simultaneous collection tasks.
- [ ] **Channels and synchronization** — coordinate workers and aggregate their results safely.
- [ ] **Graceful cancellation** — improve context cancellation and application shutdown.
- [ ] **Improved GitHub API resilience** — handle rate limits, timeouts, and transient failures more robustly.
- [ ] **Historical comparisons** — compare snapshots to show how repository statistics change over time.
- [ ] **Automated collection** — periodically collect statistics without requiring `/collect`.
- [ ] **Configuration improvements** — validate configuration and improve startup error reporting.
- [ ] **Testing** — add tests for core business logic and error handling.
- [ ] **Documentation improvements** — document database schema, setup, and operational details.

## Learning Goals

This project is also a practical learning exercise focused on:

- Go interfaces and package organization
- Dependency injection and separation of responsibilities
- HTTP clients and external API integration
- PostgreSQL and `database/sql`
- Context propagation and cancellation
- Goroutines, channels, and `sync`
- Worker pool patterns
- Error handling and structured logging

## License

No license has been added yet. Unless a license is included in the repository, the project should not be assumed to be open source under a specific license.
