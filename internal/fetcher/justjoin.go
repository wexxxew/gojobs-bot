package fetcher

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gojobs-bot/internal/model"
)

// JustJoin — https://api.justjoin.it/v2/user-panel/offers/by-cursor
// Категория 20 = Go. Уровень опыта фильтруется на стороне API.
type JustJoin struct {
	Experience []string // junior, mid, senior
}

const jjGoCategory = "20"

type jjOffer struct {
	GUID            string    `json:"guid"`
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	ExperienceLevel string    `json:"experienceLevel"`
	WorkplaceType   string    `json:"workplaceType"`
	City            string    `json:"city"`
	CompanyName     string    `json:"companyName"`
	PublishedAt     time.Time `json:"publishedAt"`
	RequiredSkills  []string  `json:"requiredSkills"`
	EmploymentTypes []struct {
		From     *float64 `json:"from"`
		To       *float64 `json:"to"`
		Currency string   `json:"currency"`
	} `json:"employmentTypes"`
}

type jjResponse struct {
	Data []jjOffer `json:"data"`
	Meta struct {
		Next struct {
			Cursor *int `json:"cursor"`
		} `json:"next"`
		TotalItems int `json:"totalItems"`
	} `json:"meta"`
}

func (JustJoin) Name() string { return "justjoin" }

func (j JustJoin) Fetch(ctx context.Context) ([]model.Vacancy, error) {
	var out []model.Vacancy
	from := 0
	for page := 0; page < 10; page++ { // защита от бесконечного цикла
		q := url.Values{}
		q.Add("categories[]", jjGoCategory)
		for _, e := range j.Experience {
			q.Add("experienceLevels[]", e)
		}
		if from > 0 {
			q.Set("from", strconv.Itoa(from))
		}

		var resp jjResponse
		u := "https://api.justjoin.it/v2/user-panel/offers/by-cursor?" + q.Encode()
		if err := getJSON(ctx, u, &resp); err != nil {
			return nil, err
		}

		for _, o := range resp.Data {
			location := o.City
			if o.WorkplaceType == "remote" {
				location = "Remote (EU/PL)"
			}
			out = append(out, model.Vacancy{
				ID:        "justjoin:" + o.GUID,
				Source:    "justjoin",
				Title:     o.Title + " [" + o.ExperienceLevel + "]",
				Company:   o.CompanyName,
				Location:  location,
				Salary:    jjSalary(o),
				URL:       "https://justjoin.it/job-offer/" + o.Slug,
				Published: o.PublishedAt,
				Tags:      o.RequiredSkills,
			})
		}

		if resp.Meta.Next.Cursor == nil || len(resp.Data) == 0 {
			break
		}
		from = *resp.Meta.Next.Cursor
	}
	return out, nil
}

func jjSalary(o jjOffer) string {
	for _, e := range o.EmploymentTypes {
		if e.From != nil && e.To != nil {
			return fmt.Sprintf("%.0f–%.0f %s/мес", *e.From, *e.To, e.Currency)
		}
	}
	return ""
}
