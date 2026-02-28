# LongCat Web API Wrapper

A lightweight compatibility layer that exposes LongCat capabilities through OpenAI- and Claude-compatible APIs, with a built-in dark admin panel, account pool management, and request logs.

## Overview

This project helps you plug LongCat into existing OpenAI/Claude clients with minimal changes.

In short, it acts as a protocol bridge:

- Exposes standard endpoints: `/v1/chat/completions`, `/v1/messages`
- Forwards requests to LongCat upstream APIs
- Manages multiple LongCat accounts (cookies) from the admin UI
- Adds routing strategy, logs, token stats, auth, and CORS

> Video generation is currently disabled (both API behavior and admin examples) for stability.

## Key Features

- OpenClaw-oriented adaptation: strips noisy "untrusted metadata" render blocks before forwarding
- Strong tool-call compatibility: when `tools` is provided, the wrapper enforces `tool_calls` JSON output to improve skill/tool invocation reliability in web clients
- OpenAI-compatible endpoint: `POST /v1/chat/completions`
- Claude-compatible endpoint: `POST /v1/messages`
- Streaming and non-streaming responses
- Capability modes: chat, search, thinking, all-in-one, image generation, deep research
- Image result rendering (including 4-image 2x2 layout output)
- Admin account pool operations (add/edit/enable/disable/test)
- Account onboarding supports both:
  - paste full Cookie string for one-click auto parsing
  - manually fill key cookie fields
- Account strategies: `average` (conversation-balanced) / `sequential` (failover)
- Dashboard metrics:
  - Total accounts
  - Healthy accounts
  - Abnormal accounts
  - Request count
  - Total tokens
- Separate logs menu in admin UI
- One-click config export/import
- Hot-reload config + persisted storage (admin changes do not require restart and survive restart)
- Optional wrapper-level API key protection
- CORS configuration + hot-reload config file

## Project Structure

```text
.
├── admin/                  # Admin pages and admin APIs
├── api/                    # OpenAI/Claude adapters and upstream handling
├── config/                 # Config model/store/cookie parser
├── conversation/           # Conversation-account binding
├── logging/                # Request log store
├── types/                  # Shared types
├── main.go                 # Entry point
├── Dockerfile
└── docker-compose.yml
```

## Requirements

- Go 1.21+
- Network access to LongCat
- At least one valid LongCat account cookie (recommended via admin UI)

## Quick Start

### 1) Run locally

```bash
go mod tidy
go run .
```

Default bind: `0.0.0.0:8082`

Then open:

- Admin UI: `http://127.0.0.1:8082/admin`
- OpenAI endpoint: `http://127.0.0.1:8082/v1/chat/completions`
- Claude endpoint: `http://127.0.0.1:8082/v1/messages`

On first startup, `./data/config.json` is created and a default admin password is printed in the terminal.

### 2) Run with Docker

```bash
docker compose up -d --build
```

Default port mapping: `8082:8082`

Persistent data directory: `./data`

## Deployment Guide

### Single-host deployment (recommended)

1. Build

```bash
go build -o longcat-web-api .
```

2. Start

```bash
mkdir -p data
./longcat-web-api
```

3. Initialize from admin UI

- Open `/admin`
- Log in with the printed default password
- Change password immediately
- Add at least one LongCat account

### Reverse proxy (Nginx/Caddy)

Proxy both `/v1/*` and `/admin*` to this service.

Recommended proxy settings:

- Forward `X-Forwarded-Proto` and `X-Forwarded-Host`
- Disable response buffering for SSE/streaming
- Use longer upstream timeouts

The admin page already supports proxy-aware scheme/host display.

## Configuration

Config file: `./data/config.json`

The service hot-reloads config updates automatically.

### Main config fields

| Field | Description | Default |
|---|---|---|
| `serverPort` | Service port | `8082` |
| `bindAddr` | Bind address | `0.0.0.0` |
| `corsAllowOrigins` | CORS allow origin(s) | `*` |
| `upstreamApiKey` | API key required by this wrapper (optional) | empty |
| `longcat.apiUrl` | LongCat chat endpoint | `https://longcat.chat/api/v1/chat-completion-V2` |
| `longcat.sessionUrl` | LongCat session endpoint | `https://longcat.chat/api/v1/session-create` |
| `longcat.timeoutSeconds` | Upstream timeout in seconds | `30` |
| `strategy.type` | Account strategy | `average` |

### Environment variables

| Variable | Description |
|---|---|
| `SERVER_PORT` | Override port |
| `BIND_ADDR` | Override bind address |
| `CORS_ALLOW_ORIGINS` | Override CORS |
| `UPSTREAM_API_KEY` | Override wrapper API key |
| `LONGCAT_API_URL` | Override LongCat API URL |
| `LONGCAT_SESSION_URL` | Override session URL |
| `LONGCAT_MODEL` | Optional default model for session creation |
| `LONGCAT_AGENT_ID` | Optional default agent for session creation |
| `TIMEOUT_SECONDS` | Default timeout |
| `VERBOSE` | `true` enables verbose logs |

> For security and operations, managing account cookies in admin UI is recommended over hardcoding env vars.

## Usage

### Admin workflow

1. Login at `/admin`
2. Add account(s) in **Account Management**
3. Configure strategy/key in **Settings**
4. Copy/test endpoints in **API Endpoints**
5. Monitor requests in **Request Logs**

### Model routing rules

The `model` value is used for behavior routing:

- contains `thinking` -> reasoning mode
- contains `search` -> web/search mode
- contains both `search` and `thinking` -> all-in-one mode (search + reasoning)
- contains `deepresearch` or `deep-research` -> deep research agent
- contains `image` or `draw` -> image generation agent
- contains `video` -> rejected (video disabled)

### OpenAI-compatible request

#### OpenClaw / tool-calling example

When `tools` is present, the wrapper injects a compact system instruction to force `tool_calls` JSON output, improving real tool invocation reliability in web clients.

```bash
curl http://127.0.0.1:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "LongCat-Search-Thinking",
    "messages": [
      {"role": "user", "content": "Check today weather in Shanghai"}
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get weather by city",
          "parameters": {
            "type": "object",
            "properties": {
              "city": {"type": "string"}
            },
            "required": ["city"]
          }
        }
      }
    ],
    "stream": false
  }'
```

Important response fields (snippet):

```json
{
  "choices": [
    {
      "finish_reason": "tool_calls",
      "message": {
        "role": "assistant",
        "tool_calls": [
          {
            "type": "function",
            "function": {
              "name": "get_weather",
              "arguments": "{\"city\":\"Shanghai\"}"
            }
          }
        ]
      }
    }
  ]
}
```

#### Non-streaming

```bash
curl http://127.0.0.1:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "LongCat-Flash",
    "messages": [
      {"role": "user", "content": "Hello"}
    ],
    "stream": false
  }'
```

#### Streaming

```bash
curl http://127.0.0.1:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "LongCat-Search-Thinking",
    "messages": [
      {"role": "user", "content": "Summarize this week in AI"}
    ],
    "stream": true
  }'
```

#### Image generation

```bash
curl http://127.0.0.1:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "LongCat-Image",
    "messages": [
      {"role": "user", "content": "Draw a cyberpunk cat"}
    ],
    "stream": true
  }'
```

### Claude-compatible request

```bash
curl http://127.0.0.1:8082/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3",
    "max_tokens": 1000,
    "messages": [
      {"role": "user", "content": "Write a small Go example"}
    ],
    "stream": false
  }'
```

### Use in any OpenAI client

- Base URL: `http://<your-host>:8082/v1`
- API Key: any string (or your configured `upstreamApiKey`)
- Model: based on routing rules above

## Security Recommendations

- Change default admin password immediately
- Set `upstreamApiKey` in production
- Put admin/API behind reverse proxy and access controls
- Never commit `data/config.json` to a public repository

## Troubleshooting

### No available account / request fails immediately

- Verify at least one enabled account exists
- Use account test in admin to verify cookie validity

### `401 unauthorized`

- If `upstreamApiKey` is configured, send one of:
  - `Authorization: Bearer <key>`
  - `X-Api-Key: <key>`

### Image rendering/layout issues

- The wrapper outputs image markdown and 2x2 for four images
- Final rendering still depends on your client markdown renderer

### Admin shows `http` behind reverse proxy

- Ensure proxy forwards `X-Forwarded-Proto` and `X-Forwarded-Host`

## Development

```bash
go mod tidy
go test ./...
go run .
```

Verbose logs:

```bash
VERBOSE=true go run .
```

## License

MIT License
