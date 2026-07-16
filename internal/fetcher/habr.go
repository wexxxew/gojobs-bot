package fetcher

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"gojobs-bot/internal/model"
)

// Habr — RSS Хабр Карьеры: https://career.habr.com/vacancies/rss?q=golang
// Замена hh.ru, который с некоторых пор требует токен приложения.
type Habr struct {
	Query string
}

var (
	habrTitleRe   = regexp.MustCompile(`^Требуется «(.+?)»\s*(.*)$`)
	habrCompanyRe = regexp.MustCompile(`Компания «([^»]+)»`)
)

func (Habr) Name() string { return "habr" }

func (h Habr) Fetch(ctx context.Context) ([]model.Vacancy, error) {
	q := h.Query
	if q == "" {
		q = "golang"
	}
	feedURL := "https://career.habr.com/vacancies/rss?type=all&q=" + url.QueryEscape(q)

	fp := gofeed.NewParser()
	fp.UserAgent = userAgent
	feed, err := fp.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, err
	}

	var out []model.Vacancy
	for _, item := range feed.Items {
		title := item.Title
		if m := habrTitleRe.FindStringSubmatch(title); m != nil {
			title = strings.TrimSpace(m[1] + " " + m[2])
		}
		company := ""
		if m := habrCompanyRe.FindStringSubmatch(item.Description); m != nil {
			company = m[1]
		}
		published := time.Now()
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}
		id := item.GUID
		if id == "" {
			id = item.Link
		}
		out = append(out, model.Vacancy{
			ID:          "habr:" + id,
			Source:      "habr",
			Title:       title,
			Company:     company,
			Location:    "РФ / удалённо",
			URL:         item.Link,
			Published:   published,
			Description: item.Description, // содержит хэштеги #senior/#middle — нужно фильтру
		})
	}
	return out, nil
}
