package bot

import (
	"fmt"
	"github-watcher/internal/service"
	"time"
)

func formatTime(t time.Time) string {
	return t.UTC().Format("02.01.2006 15:04:05 UTC")
}

func formatLastCommit(t *time.Time) string {
	if t == nil {
		return "нет коммитов"
	}
	return formatTime(*t)
}

func formatRepositoryStats(stat service.RepositoryStats) string {
	if stat.Err != nil {
		return fmt.Sprintf("Не удалось получить снимки репозитория %s.\n", stat.Repository.FullName)
	}

	if stat.Snapshot == nil {
		return fmt.Sprintf("Репозиторий %s не имеет снимков.\n", stat.Repository.FullName)
	}

	snapshot := stat.Snapshot

	return fmt.Sprintf(
		"📦 %s\n"+
			"⭐ Stars: %d\n"+
			"🍴 Forks: %d\n"+
			"🐛 Open issues: %d\n"+
			"🔀 Open PRs: %d\n"+
			"💻 Commits за 7 дней: %d\n"+
			"📄 README: %t\n"+
			"🕒 Последний сбор: %s\n"+
			"🕒 Последний коммит: %s\n"+
			"━━━━━━━━━━━━━━━━━━━━\n",
		stat.Repository.FullName,
		snapshot.Stars,
		snapshot.Forks,
		snapshot.OpenIssues,
		snapshot.OpenPullRequests,
		snapshot.CommitsLast7Days,
		snapshot.HasReadme,
		formatTime(snapshot.CollectedAt),
		formatLastCommit(snapshot.LastCommitAt),
	)
}
