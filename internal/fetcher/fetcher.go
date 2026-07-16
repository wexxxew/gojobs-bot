package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gojobs-bot/internal/model"
)

const userAgent = "gojobs-bot/1.0 (personal job aggregator)"

// Fetcher — единственный интерфейс, который нужно реализовать,
// чтобы добавить новый источник вакансий.
type Fetcher interface {
	Name() string
	Fetch(ctx context.Context) ([]model.Vacancy, error)
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
