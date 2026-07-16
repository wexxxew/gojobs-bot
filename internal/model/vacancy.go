package model

import "time"

// Vacancy — единый формат вакансии, к которому каждый источник
// приводит свои данные.
type Vacancy struct {
	ID          string    // уникальный ID: "источник:внешний_id", ключ дедупликации
	Source      string    // remoteok, justjoin, golangprojects, habr
	Title       string
	Company     string
	Location    string
	Salary      string // как есть, строкой — форматы у всех источников разные
	URL         string
	Published   time.Time
	Description string   // сырое описание, если источник его отдаёт — нужно фильтру
	Tags        []string // теги источника (junior/senior и т.п.) — тоже для фильтра
}
