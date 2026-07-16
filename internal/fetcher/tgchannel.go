package fetcher

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gojobs-bot/internal/model"
)

// TGChannel читает публичный Telegram-канал через веб-превью t.me/s/<канал>.
// Не требует ни бот-токена, ни MTProto-сессии: превью отдаёт последние
// ~20 постов обычным HTML. Работает только для каналов с включённым превью.
type TGChannel struct {
	Channel  string
	Keywords []string // для смешанных каналов: пост должен упоминать одно из слов
}

var (
	tgPostRe = regexp.MustCompile(`data-post="([^"]+)"`)
	tgTextRe = regexp.MustCompile(`(?s)class="tgme_widget_message_text[^"]*"[^>]*>(.*?)</div>`)
	tgTimeRe = regexp.MustCompile(`datetime="([^"]+)"`)
	tgBrRe   = regexp.MustCompile(`(?i)<br\s*/?>`)
	tgTagRe  = regexp.MustCompile(`<[^>]+>`)
)

func (t TGChannel) Name() string { return "tg:" + t.Channel }

func (t TGChannel) Fetch(ctx context.Context) ([]model.Vacancy, error) {
	page, err := t.fetchHTML(ctx)
	if err != nil {
		return nil, err
	}

	// режем страницу на блоки-сообщения по маркеру data-post="канал/123"
	locs := tgPostRe.FindAllStringSubmatchIndex(page, -1)
	if len(locs) == 0 {
		return nil, fmt.Errorf("t.me/s/%s: не нашёл ни одного поста (превью выключено?)", t.Channel)
	}

	var out []model.Vacancy
	for i, loc := range locs {
		post := page[loc[2]:loc[3]] // "канал/123"
		end := len(page)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		block := page[loc[0]:end]

		m := tgTextRe.FindStringSubmatch(block)
		if m == nil {
			continue // пост без текста (фото, репост)
		}
		text := cleanHTML(m[1])
		if text == "" || !t.matchesKeywords(text) {
			continue
		}

		published := time.Time{}
		if tm := tgTimeRe.FindAllStringSubmatch(block, -1); len(tm) > 0 {
			// последний datetime в блоке — время самого сообщения
			published, _ = time.Parse(time.RFC3339, tm[len(tm)-1][1])
		}

		out = append(out, model.Vacancy{
			ID:          "tg:" + post,
			Source:      "tg_" + t.Channel,
			Title:       postTitle(text),
			Location:    "см. пост",
			URL:         "https://t.me/" + post,
			Published:   published,
			Description: text,
		})
	}
	return out, nil
}

func (t TGChannel) fetchHTML(ctx context.Context) (string, error) {
	url := "https://t.me/s/" + t.Channel
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; gojobs-bot/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return string(data), err
}

func (t TGChannel) matchesKeywords(text string) bool {
	if len(t.Keywords) == 0 {
		return true
	}
	for _, k := range t.Keywords {
		// границы слова, чтобы "go" не совпадал с "django" или "года"
		re := regexp.MustCompile(`(?i)(^|[^\p{L}\d])` + regexp.QuoteMeta(k) + `([^\p{L}\d]|$)`)
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

func cleanHTML(s string) string {
	s = tgBrRe.ReplaceAllString(s, "\n")
	s = tgTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// первая непустая строка поста, без служебного «#вакансия», не длиннее 90 символов
func postTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "#вакансия"))
		line = strings.TrimLeft(line, " :,-—")
		if line == "" {
			continue
		}
		r := []rune(line)
		if len(r) > 90 {
			return string(r[:90]) + "…"
		}
		return line
	}
	return "(без заголовка)"
}
