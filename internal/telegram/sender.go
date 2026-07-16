package telegram

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gojobs-bot/internal/model"
)

// Sender шлёт вакансии в личку через Bot API.
// Без токена работает в dry-run режиме: печатает в stdout —
// удобно тестировать без бота.
type Sender struct {
	Token  string
	ChatID string
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func (s *Sender) DryRun() bool { return s.Token == "" || s.ChatID == "" }

func (s *Sender) Send(ctx context.Context, v model.Vacancy) error {
	if s.DryRun() {
		fmt.Println(plainText(v))
		return nil
	}

	payload := url.Values{
		"chat_id":                  {s.ChatID},
		"text":                     {htmlText(v)},
		"parse_mode":               {"HTML"},
		"disable_web_page_preview": {"true"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+s.Token+"/sendMessage",
		strings.NewReader(payload.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("telegram: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func htmlText(v model.Vacancy) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🟢 <b>%s</b>\n", html.EscapeString(v.Title))

	line2 := []string{}
	if v.Company != "" {
		line2 = append(line2, "🏢 "+html.EscapeString(v.Company))
	}
	if v.Location != "" {
		line2 = append(line2, "📍 "+html.EscapeString(v.Location))
	}
	if len(line2) > 0 {
		b.WriteString(strings.Join(line2, " · ") + "\n")
	}
	if v.Salary != "" {
		b.WriteString("💰 " + html.EscapeString(v.Salary) + "\n")
	}
	fmt.Fprintf(&b, "🔗 %s\n#%s", html.EscapeString(v.URL), v.Source)
	return b.String()
}

func plainText(v model.Vacancy) string {
	return fmt.Sprintf("[%s] %s | %s | %s | %s | %s",
		v.Source, v.Title, v.Company, v.Location, v.Salary, v.URL)
}
