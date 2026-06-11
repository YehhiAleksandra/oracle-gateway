# oracle-gateway

HTTP gateway for [OpenRouter](https://openrouter.ai/) with **model fallback**, **429 retry**, and **timeouts** — extracted from the [Зинаида](https://github.com/YehhiAleksandra/digital-oracle) bot.

Keeps LLM keys and retry logic in one place so Telegram bots stay thin.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/ready` | Checks `OPENAI_API_KEY` |
| GET | `/v1/ping` | Quick LLM ping |
| POST | `/v1/complete` | Chat completion with task-based model routing |
| POST | `/v1/vision` | Vision completion (palmistry, graphology) |

### POST `/v1/complete`

```json
{
  "system": "You are Zinaida...",
  "user": "Birth date 17.05.1994",
  "max_tokens": 900,
  "task": "numerology"
}
```

`task` selects the model chain: `tarot`, `runes`, `numerology`, `astrology`, `horoscope`, `palmistry`, `graphology`, `ping`.

Override per task via env: `MODEL_TASK_TAROT=model1,model2` and `MODEL_VISION_TASK_PALMISTRY=vision1,vision2`.

Response:

```json
{
  "text": "...",
  "model": "openai/gpt-oss-120b:free",
  "elapsed_ms": 15200
}
```

## Run locally

```bash
cp .env.example .env
go run .
curl -s localhost:8080/health
```

## Docker

```bash
docker compose up --build
```

## Stack

- Go 1.22
- OpenRouter-compatible API
- Docker + GitHub Actions CI

## Related

- [digital-oracle](https://github.com/YehhiAleksandra/digital-oracle)
- [astro-core](https://github.com/YehhiAleksandra/astro-core)

## License

MIT

---

<p align="center"><sub><a href="https://github.com/YehhiAleksandra">@YehhiAleksandra</a> · Yehhi Aleksandra · Telegram bots & automation</sub></p>