# GitHub Watcher

A Telegram bot for monitoring GitHub repositories and tracking changes in their statistics over time.

Built with Go, the GitHub REST API, and PostgreSQL, the application collects repository metrics on demand and stores historical snapshots for later analysis.

## Features

- **Repository Management** — add and list GitHub repositories to track.
- **Statistics Collection** — retrieve stars, forks, open issues, open pull requests, README availability, and commit activity.
- **Historical Snapshots** — persist collected metrics in PostgreSQL to keep a history of repository statistics.
- **Concurrent Processing** — collect statistics for multiple repositories using Go concurrency primitives.
- **Telegram Interface** — manage repositories and access collected statistics through bot commands.
- **Containerized Deployment** — run the bot and PostgreSQL using Docker Compose.

## Tech Stack

| Technology | Purpose |
|---|---|
| Go | Application logic and concurrent processing |
| PostgreSQL | Persistent storage for repositories and snapshots |
| GitHub REST API | Repository metadata and activity |
| Telegram Bot API | User interaction |
| Docker Compose | Application deployment and database orchestration |

## Architecture

## Project Structure

```text
.
├── Dockerfile
├── PLAN.md
├── README.md
├── compose.yaml
├── go.mod
├── go.sum
├── internal
│   ├── bot
│   │   └── handlers.go
│   ├── github
│   │   ├── client.go
│   │   ├── commits.go
│   │   ├── openPullRequests.go
│   │   └── repositories.go
│   ├── logger
│   │   └── logger.go
│   ├── models
│   │   ├── repository.go
│   │   └── repositorySnapshot.go
│   ├── postgres
│   │   ├── postgres.go
│   │   └── repositories.go
│   ├── service
│   │   └── service.go
│   ├── storage
│   │   └── storage.go
│   └── tracker
│       └── tracker.go
├── main.go
└── migrations
    └── scheme.sql
```

The application separates external API access, business logic, persistence, and user interaction.

- `internal/bot` — processes Telegram commands.
- `internal/github` — encapsulates GitHub API requests.
- `internal/service` — handles repository management and validation.
- `internal/tracker` — collects metrics and saves snapshots.
- `internal/storage` — defines the persistence contract.
- `internal/postgres` — implements persistence using PostgreSQL.
- `internal/logger` — configures application logging.
- `models` — defines repository and snapshot data structures.
- `migrations` — contains the initial database schema.

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

Create your local environment file:

```bash
cp .env.example .env
```

Configure the variables in `.env`:

```dotenv
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
POSTGRES_PASSWORD=your_database_password
DATABASE_URL=postgres://watcher:your_database_password@db:5432/github_watcher
```

Replace the placeholders with your own credentials. The password in `DATABASE_URL` must match `POSTGRES_PASSWORD`.

### 3. Run the application

```bash
docker compose up --build -d
```

Docker Compose starts PostgreSQL, initializes the schema on the first database startup, and starts the bot after PostgreSQL becomes healthy.

Open your bot in Telegram and send `/start`.

## Bot Commands

| Command                 | Description                          |
|-------------------------|--------------------------------------|
| `/start`                | Start interacting with the bot       |
| `/help`                 | Show available commands              |
| `/add owner/repository` | Add a repository to track            |
| `/delete`               | Delete a repository from track       |
| `/list`                 | List tracked repositories            |
| `/collect`              | Collect and save new statistics      |
| `/stats`                | View previously collected statistics |

## Database

The application stores tracked repositories and their historical statistics in PostgreSQL.

- `repositories` stores repository metadata.
- `repository_snapshots` stores collected metrics and collection timestamps.

A foreign key with cascading deletion keeps repository records and their snapshots consistent. A Docker-managed volume preserves database data across container restarts.

The initial schema is defined in `migrations/scheme.sql` and is applied automatically when PostgreSQL initializes an empty data directory.
