package fetcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gojobs-bot/internal/model"
)

// RemoteOK — https://remoteok.com/api
// API отдаёт весь свежий список одним JSON-массивом; фильтр ?tag= на их
// стороне не работает, поэтому отбираем Go-вакансии по тегам сами.
type RemoteOK struct{}

type remoteOKItem struct {
	Slug      string   `json:"slug"`
	Position  string   `json:"position"`
	Company   string   `json:"company"`
	Location  string   `json:"location"`
	URL       string   `json:"url"`
	Date      string   `json:"date"`
	Tags      []string `json:"tags"`
	SalaryMin int      `json:"salary_min"`
	SalaryMax int      `json:"salary_max"`
}

func (RemoteOK) Name() string { return "remoteok" }

func (RemoteOK) Fetch(ctx context.Context) ([]model.Vacancy, error) {
	var items []remoteOKItem
	if err := getJSON(ctx, "https://remoteok.com/api", &items); err != nil {
		return nil, err
	}

	var out []model.Vacancy
	for _, it := range items {
		// первый элемент массива — юридическая заглушка без position
		if it.Position == "" || !hasGoTag(it.Tags) {
			continue
		}
		published, _ := time.Parse(time.RFC3339, it.Date)
		location := it.Location
		if location == "" {
			location = "Remote"
		}
		out = append(out, model.Vacancy{
			ID:        "remoteok:" + it.Slug,
			Source:    "remoteok",
			Title:     it.Position,
			Company:   it.Company,
			Location:  location,
			Salary:    usdRange(it.SalaryMin, it.SalaryMax),
			URL:       it.URL,
			Published: published,
			Tags:      it.Tags,
		})
	}
	return out, nil
}

func hasGoTag(tags []string) bool {
	for _, t := range tags {
		t = strings.ToLower(t)
		if t == "golang" || t == "go" {
			return true
		}
	}
	return false
}

func usdRange(min, max int) string {
	if min == 0 && max == 0 {
		return ""
	}
	return fmt.Sprintf("$%dk–%dk/год", min/1000, max/1000)
}
