package fetcher

import (
	"context"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"gojobs-bot/internal/model"
)

// GolangProjects — RSS-лента https://www.golangprojects.com/rss.xml
// Только Go-вакансии со всего мира, много удалёнки.
type GolangProjects struct{}

func (GolangProjects) Name() string { return "golangprojects" }

func (GolangProjects) Fetch(ctx context.Context) ([]model.Vacancy, error) {
	fp := gofeed.NewParser()
	fp.UserAgent = userAgent
	feed, err := fp.ParseURLWithContext("https://www.golangprojects.com/rss.xml", ctx)
	if err != nil {
		return nil, err
	}

	var out []model.Vacancy
	for _, item := range feed.Items {
		title, company := splitTitleCompany(item.Title)
		published := time.Now()
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}
		id := item.GUID
		if id == "" {
			id = item.Link
		}
		out = append(out, model.Vacancy{
			ID:          "golangprojects:" + id,
			Source:      "golangprojects",
			Title:       title,
			Company:     company,
			Location:    "см. описание",
			URL:         item.Link,
			Published:   published,
			Description: item.Description,
		})
	}
	return out, nil
}

// заголовки вида "Backend Engineer @ Acme (Remote, Europe)" —
// часть после "@" считаем компанией
func splitTitleCompany(title string) (string, string) {
	if before, after, found := strings.Cut(title, " @ "); found {
		return strings.TrimSpace(before), strings.TrimSpace(after)
	}
	return title, ""
}
