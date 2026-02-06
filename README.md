# ATS

Minimal end-to-end Alpaca paper-trading bot in Go. The project focuses on a single-symbol loop:
market data bars → deterministic SMA strategy → risk gate → paper orders, with reconciliation and
structured logging.

## Features
- Market data streaming (v2/test for stream mode, v2/iex for paper mode)
- Deterministic SMA strategy (SMA(20) on close)
- Hard risk checks (cooldown, max position, max notional, long-only, one open order)
- Paper trading via Alpaca REST API
- Decision logging to newline-delimited JSON
- Optional checkpoint state on shutdown

## Requirements
- Go 1.22+
- Alpaca API credentials for paper mode (env vars `APCA_API_KEY_ID`, `APCA_API_SECRET_KEY`)

## .env support
If a `.env` file is present in the repo root, the bot will load missing environment variables from it.
Existing environment variables always take precedence.

## Run
Stream mode (no orders; uses test feed):

```bash
APCA_API_KEY_ID=your_key APCA_API_SECRET_KEY=your_secret \
  go run ./cmd/bot --mode=stream --symbol=FAKEPACA
```

Paper mode (places orders):

```bash
APCA_API_KEY_ID=your_key APCA_API_SECRET_KEY=your_secret \
  go run ./cmd/bot --mode=paper --symbol=AAPL
```

## Quick start
1) Run in stream mode (no orders, no credentials needed):

```bash
go run ./cmd/bot --mode=stream --symbol=FAKEPACA
```

2) Paper trading (requires Alpaca paper keys):

```bash
APCA_API_KEY_ID=your_key APCA_API_SECRET_KEY=your_secret \
  go run ./cmd/bot --mode=paper --symbol=AAPL
```

3) Optional: LLM strategy (Ollama running locally, plus model set):

```bash
LLM_MODEL=llama3 \
APCA_API_KEY_ID=your_key APCA_API_SECRET_KEY=your_secret \
  go run ./cmd/bot --mode=paper --strategy=llm --symbol=AAPL
```

4) Optional: use `config.json` to avoid flags (defaults apply if missing):

```json
{
  "mode": "paper",
  "strategy": "llm",
  "maxQty": 1
}
```

## Configuration flags
- `--mode` (stream|paper)
- `--symbol` (default: FAKEPACA in stream mode, AAPL in paper mode)
- `--feed` (default: test in stream mode, iex in paper mode)
- `--strategy` (random_noise|mean_reversion|sma|llm)
- `--config` (optional path to JSON config file; defaults to `./config.json` if present)

## Environment Variables

- `APCA_API_KEY_ID` - Alpaca API key ID (required in paper mode)
- `APCA_API_SECRET_KEY` - Alpaca API secret key (required in paper mode)
- `LOG_FORMAT` - Log output format: `json` for JSON (recommended for containers/production) or omit for pretty text (local development)
- `--bars-window` (default: 50)
- `--sma-window` (default: 20)
- `--max-qty` (default: 1)
- `--max-notional` (default: 200)
- `--cooldown` (default: 120s)
- `--reconcile-interval` (default: 10s)
- `--kill-switch` (default: false)
- `--extended-hours` (default: false)
- `--order-type` (default: market)
- `--time-in-force` (default: day)
- `--decisions-path` (default: decisions.ndjson)
- `--checkpoint-path` (default: checkpoint.json)
- `--paper-base-url` (default: https://paper-api.alpaca.markets)

## LLM configuration (environment variables)
- `LLM_BASE_URL` (default: http://localhost:11434 for Ollama)
- `LLM_MODEL` (required for strategy=llm)
- `LLM_TIMEOUT` (default: 8s)
- `LLM_CONTEXT_PROMPT` (extra context appended into the decision prompt)
- `LLM_SYSTEM_PROMPT_PATH` (optional override for the system prompt markdown template)
- `LLM_DECISION_PROMPT_PATH` (optional override for the decision prompt markdown template)

Default prompt templates live in `internal/llm/prompts/` and can be extended by copying and
overriding via the environment variable paths.

## Configuration precedence
Defaults → JSON config file → environment variables → CLI flags.
CLI flags always win if provided.

## Development notes
This repo uses a local stub of the Alpaca SDK for offline builds/tests. To use the real SDK,
remove the `replace` directive in `go.mod` and run `go mod tidy` with network access.

## Testing
```bash
go test ./...
```

## Output
- `decisions.ndjson` records each decision cycle for replay/debugging.
- `checkpoint.json` captures position/open-order state on shutdown.
