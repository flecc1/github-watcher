# GitHub Watcher

A Telegram bot written in Go that tracks GitHub repository statistics and stores historical snapshots in PostgreSQL.

## Features

- **Repository tracking** — add GitHub repositories using the `owner/repo` format.
- **Repository management** — list tracked repositories.
- **Statistics collection** — fetch repository metrics from the GitHub API and save snapshots to PostgreSQL.
- **Historical snapshots** — preserve collected statistics over time.
- **Statistics overview** — display the latest saved snapshot for each tracked repository.
- **Concurrent collection** — process repositories using a worker pool with three workers.
- **Context-aware operations** — use timeouts and cancellation for requests and database operations.

## Telegram Commands

| Command | Description |
|---|---|
| `/start` | Start interacting with the bot |
| `/help` | Display available commands |
| `/add owner/repo` | Add a repository to track |
| `/list` | List tracked repositories |
| `/collect` | Collect and save fresh statistics |
| `/stats` | Display the latest saved statistics |

## Collected Metrics

The bot collects the following information for each repository:

- Stars and forks
- Open issues
- Open pull requests
- README availability
- Commits made during the last 7 days
- Last commit timestamp
- Snapshot collection timestamp

## Tech Stack

- **Go** — application logic and concurrency
- **PostgreSQL** — persistent storage and historical snapshots
- **Telegram Bot API** — bot interaction
- **GitHub REST API** — repository information
- **`context`** — timeouts and cancellation
- **Goroutines, channels and `sync.WaitGroup`** — concurrent collection

## Project Structure

```text
github-watcher/
├── internal/
│   ├── bot/          # Telegram command handlers
│   ├── github/       # GitHub API client
│   ├── postgres/     # PostgreSQL implementation
│   ├── repository/   # Repository management
│   ├── storage/      # Storage interface and errors
│   └── tracker/      # Statistics collection and snapshots
├── models/           # Domain models
├── main.go           # Application initialization
├── go.mod
└── README.md
```

## Getting Started

### Prerequisites

- Go installed
- PostgreSQL
- A Telegram bot token from [@BotFather](https://t.me/BotFather)

### 1. Clone the repository

```bash
git clone https://github.com/flecc1/github-watcher.git
cd github-watcher
```

### 2. Configure environment variables

Create a `.env` file in the project root and configure the Telegram bot token and PostgreSQL connection string.

Example:

```env
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
DATABASE_URL=postgres://user:password@localhost:5432/github_watcher?sslmode=disable
```

Use the exact environment variable names expected by the application.

### 3. Set up the database

Create a PostgreSQL database and apply the schema required by the application. Make sure the database tables and indexes match the SQL definitions used by the project.

### 4. Run the application

```bash
go mod download
go run .
```

Once the bot starts, open it in Telegram and send `/start`.

## Concurrency

The `/collect` command uses a worker pool to collect statistics from multiple repositories concurrently.

- A jobs channel distributes repositories to workers.
- Three workers process repositories independently.
- A results channel returns collection outcomes.
- A `sync.WaitGroup` tracks worker completion.
- The results channel is closed after all workers finish.
- Context cancellation and timeouts help prevent operations from running indefinitely.

The `/stats` command reads previously stored snapshots and does not initiate a new collection.

## Future Improvements

- Scheduled statistics collection
- Notifications about significant repository changes
- GitHub API rate-limit handling
- Automated tests
- Docker Compose setup