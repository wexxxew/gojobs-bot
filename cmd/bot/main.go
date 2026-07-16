package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gojobs-bot/internal/config"
	"gojobs-bot/internal/fetcher"
	"gojobs-bot/internal/filter"
	"gojobs-bot/internal/model"
	"gojobs-bot/internal/storage"
	"gojobs-bot/internal/telegram"
)

func main() {
	loadDotEnv(".env")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	seen, err := storage.Load(cfg.SeenFile)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// --- собираем вакансии со всех источников параллельно ---
	var fetchers []fetcher.Fetcher
	if cfg.Sources.RemoteOK.Enabled {
		fetchers = append(fetchers, fetcher.RemoteOK{})
	}
	if cfg.Sources.JustJoin.Enabled {
		fetchers = append(fetchers, fetcher.JustJoin{Experience: cfg.Sources.JustJoin.Experience})
	}
	if cfg.Sources.GolangProjects.Enabled {
		fetchers = append(fetchers, fetcher.GolangProjects{})
	}
	if cfg.Sources.Habr.Enabled {
		fetchers = append(fetchers, fetcher.Habr{Query: cfg.Sources.Habr.Query})
	}
	for _, ch := range cfg.Sources.TGChannels {
		fetchers = append(fetchers, fetcher.TGChannel{Channel: ch.Channel, Keywords: ch.Keywords})
	}

	var (
		mu  sync.Mutex
		all []model.Vacancy
		wg  sync.WaitGroup
	)
	for _, f := range fetchers {
		wg.Add(1)
		go func(f fetcher.Fetcher) {
			defer wg.Done()
			vs, err := f.Fetch(ctx)
			if err != nil {
				// один упавший источник не должен ронять остальные
				log.Printf("[%s] ошибка: %v", f.Name(), err)
				return
			}
			log.Printf("[%s] получено вакансий: %d", f.Name(), len(vs))
			mu.Lock()
			all = append(all, vs...)
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	// --- фильтр по уровню + дедупликация ---
	flt := filter.New(filter.Config{
		RequireInclude: cfg.Filter.RequireInclude,
		Include:        cfg.Filter.Include,
		Exclude:        cfg.Filter.Exclude,
	})
	// вакансии старше max_age_days молча помечаем как виденные:
	// это защита от потопа старыми постами при подключении нового источника
	tooOld := time.Now().AddDate(0, 0, -cfg.MaxAgeDays)

	var fresh []model.Vacancy
	for _, v := range all {
		if seen.Has(v.ID) || !flt.Match(v) {
			continue
		}
		if !v.Published.IsZero() && v.Published.Before(tooOld) {
			seen.Add(v.ID)
			continue
		}
		fresh = append(fresh, v)
	}
	sort.Slice(fresh, func(i, j int) bool { return fresh[i].Published.Before(fresh[j].Published) })

	// первый запуск: seen.json пуст, отправляем только самые свежие,
	// остальные молча помечаем как виденные — иначе прилетит вся история
	if seen.Empty() && len(fresh) > cfg.MaxPerRun {
		for _, v := range fresh[:len(fresh)-cfg.MaxPerRun] {
			seen.Add(v.ID)
		}
		fresh = fresh[len(fresh)-cfg.MaxPerRun:]
	}
	// обычный запуск: лимит на отправку, остальные уйдут в следующий раз
	if len(fresh) > cfg.MaxPerRun {
		fresh = fresh[:cfg.MaxPerRun]
	}

	// --- отправка ---
	sender := &telegram.Sender{
		Token:  os.Getenv("TELEGRAM_TOKEN"),
		ChatID: os.Getenv("TELEGRAM_CHAT_ID"),
	}
	if sender.DryRun() {
		log.Println("TELEGRAM_TOKEN/TELEGRAM_CHAT_ID не заданы — dry-run, печатаю в stdout")
	}

	sent := 0
	for _, v := range fresh {
		if err := sender.Send(ctx, v); err != nil {
			log.Printf("отправка: %v", err)
			break // не помечаем как seen — уйдёт в следующий запуск
		}
		seen.Add(v.ID)
		sent++
		if !sender.DryRun() {
			time.Sleep(1100 * time.Millisecond) // лимит Telegram: ~1 сообщение/сек
		}
	}

	if err := seen.Save(); err != nil {
		log.Fatalf("сохранение %s: %v", cfg.SeenFile, err)
	}
	log.Printf("готово: новых вакансий отправлено — %d", sent)
}

// loadDotEnv — крошечный загрузчик .env, чтобы не тянуть зависимость.
// Переменные из окружения имеют приоритет над файлом.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // файла нет — значит, работаем от окружения (CI)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
