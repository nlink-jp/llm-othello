# llm-othello

Browser-based Othello game where you play against a local LLM.

- **You** play as Black, **LLM** plays as White
- LLM move decisions happen server-side — the client only handles board display and game logic
- Works with any OpenAI-compatible API (LM Studio, Ollama, OpenAI, etc.)

## Setup

**Prerequisites:** Go 1.24+

```bash
git clone https://github.com/nlink-jp/llm-othello
cd llm-othello
go mod tidy
```

Copy `config.toml.example` to `config.toml` and edit to point at your LLM server:

```sh
cp config.toml.example config.toml
```

```toml
[llm]
base_url    = "http://localhost:1234"  # LM Studio default
model       = "your-model-name"
api_key     = ""
temperature = 0.0
```

## Usage

```bash
go run . [-config config.toml]
```

Then open http://localhost:8080 in your browser.

Or build first:

```bash
make build
./llm-othello
```

## Configuration

| Key | Default | Description |
|-----|---------|-------------|
| `llm.base_url` | `http://localhost:1234` | OpenAI-compatible API base URL |
| `llm.model` | `local-model` | Model name to request |
| `llm.api_key` | `""` | API key (leave empty for local LLM) |
| `llm.temperature` | `0.0` | Sampling temperature (0 = deterministic) |
| `server.port` | `8080` | HTTP port |

## How it works

```
Browser                    Go server               LLM
  |                            |                    |
  |-- POST /api/move -------->|                    |
  |   {board, validMoves}      |-- chat/completions->|
  |                            |<-- chosen move -----|
  |<-- {row, col} ------------|                    |
```

The server builds a text prompt from the board state and valid moves, calls the LLM, parses the JSON response, validates the move is legal, and returns it. If the LLM returns an invalid move or fails, a random legal move is chosen as fallback.
