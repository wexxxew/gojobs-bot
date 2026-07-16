package filter

import (
	"strings"

	"gojobs-bot/internal/model"
)

type Config struct {
	// RequireInclude: true — слать только вакансии со словами из Include;
	// false — слать всё, где нет стоп-слов (по умолчанию: junior-вакансии
	// редко подписаны словом "junior", а senior почти всегда подписаны).
	RequireInclude bool
	Include        []string
	Exclude        []string
}

type Filter struct {
	cfg Config
}

func New(cfg Config) *Filter {
	return &Filter{cfg: cfg}
}

func (f *Filter) Match(v model.Vacancy) bool {
	text := strings.ToLower(v.Title + " " + v.Description + " " + strings.Join(v.Tags, " "))

	for _, w := range f.cfg.Exclude {
		if strings.Contains(text, strings.ToLower(w)) {
			return false
		}
	}
	if !f.cfg.RequireInclude {
		return true
	}
	for _, w := range f.cfg.Include {
		if strings.Contains(text, strings.ToLower(w)) {
			return true
		}
	}
	return false
}
