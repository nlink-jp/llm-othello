# llm-othello

ローカル LLM と対戦する、ブラウザ版のオセロ。

- **あなた**が黒、**LLM** が白を持つ
- LLM の着手判断はサーバー側で行う — クライアントは盤面の表示とゲームロジックだけを担う
- OpenAI 互換 API であれば何でも動く（LM Studio、Ollama、OpenAI など）

## セットアップ

**前提条件:** Go 1.24 以降

```bash
git clone https://github.com/nlink-jp/llm-othello
cd llm-othello
go mod tidy
```

`config.toml.example` を `config.toml` にコピーし、自分の LLM サーバーを指すように編集する。

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

## 使い方

```bash
go run . [-config config.toml]
```

続いて、ブラウザで http://localhost:8080 を開く。

先にビルドしてもよい。

```bash
make build
./dist/llm-othello
```

## 設定

| キー | 既定値 | 説明 |
|-----|---------|-------------|
| `llm.base_url` | `http://localhost:1234` | OpenAI 互換 API のベース URL |
| `llm.model` | `local-model` | 要求するモデル名 |
| `llm.api_key` | `""` | API キー（ローカル LLM では空のままにする） |
| `llm.temperature` | `0.0` | サンプリング温度（0 = 決定的） |
| `server.port` | `8080` | HTTP ポート |

## 仕組み

```
Browser                    Go server               LLM
  |                            |                    |
  |-- POST /api/move -------->|                    |
  |   {board, validMoves}      |-- chat/completions->|
  |                            |<-- chosen move -----|
  |<-- {row, col} ------------|                    |
```

サーバーは盤面の状態と合法手からテキストのプロンプトを組み立て、LLM を呼び、JSON の応答を解析し、その着手が合法であることを検証して返す。LLM が不正な手を返したときや失敗したときは、フォールバックとして合法手をランダムに 1 つ選ぶ。
