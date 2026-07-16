# gojobs-bot

Telegram-бот, который собирает junior/middle Go-вакансии из нескольких
источников и присылает новые тебе в личку. Работает без сервера — по
расписанию в GitHub Actions, полностью бесплатно.

## Источники

| Источник | Что даёт | Как подключён |
|---|---|---|
| [Golangprojects](https://www.golangprojects.com) | Go-вакансии по всему миру, много удалёнки | RSS |
| [RemoteOK](https://remoteok.com) | глобальная удалёнка | JSON API |
| [Justjoin.it](https://justjoin.it) | Польша/ЕС, фильтр по уровню на стороне API | JSON API |
| [Хабр Карьера](https://career.habr.com) | русскоязычный рынок | RSS |

Добавить источник = реализовать интерфейс из одного метода
(`internal/fetcher/fetcher.go`):

```go
type Fetcher interface {
    Name() string
    Fetch(ctx context.Context) ([]model.Vacancy, error)
}
```

## Как это работает

```
GitHub Actions (cron каждые 30 мин)
  └─ go run ./cmd/bot
       ├─ 4 источника опрашиваются параллельно (горутины)
       ├─ фильтр по стоп-словам (senior/lead/... отсекаются)
       ├─ дедупликация по seen.json (коммитится обратно в репо)
       └─ новые вакансии уходят в Telegram
```

Без токена бот работает в dry-run режиме и печатает вакансии в консоль —
удобно проверять фильтры.

## Запуск локально

```
cp .env.example .env   # вписать TELEGRAM_TOKEN и TELEGRAM_CHAT_ID
go run ./cmd/bot
```

- Токен: написать @BotFather команду `/newbot`.
- Свой chat_id: написать что-нибудь боту @userinfobot.
- После создания бота нужно один раз нажать Start у своего бота,
  иначе он не сможет писать первым.

## Деплой (бесплатно, GitHub Actions)

1. Залить репозиторий на GitHub (публичный — тогда минуты Actions не ограничены).
2. Settings → Secrets and variables → Actions → добавить секреты
   `TELEGRAM_TOKEN` и `TELEGRAM_CHAT_ID`.
3. Всё. Workflow `check vacancies` запускается каждые 30 минут,
   первый раз можно дёрнуть руками: Actions → check vacancies → Run workflow.

Нюанс: если в репозитории 60 дней нет коммитов, GitHub усыпляет расписание.
Коммиты `seen.json` от самого бота считаются активностью, так что на
практике это не проблема, пока появляются новые вакансии.

## Настройка

Всё в `config.yaml`: список стоп-слов, уровни опыта для Justjoin,
лимит сообщений за запуск, включение/выключение источников.
