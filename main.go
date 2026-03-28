package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

//go:embed index.html script.js style.css
var static embed.FS

// ---- Config ----------------------------------------------------------------

type Config struct {
	LLM    LLMConfig    `toml:"llm"`
	Server ServerConfig `toml:"server"`
}

type LLMConfig struct {
	BaseURL     string  `toml:"base_url"`
	Model       string  `toml:"model"`
	APIKey      string  `toml:"api_key"`
	Temperature float64 `toml:"temperature"`
}

type ServerConfig struct {
	Port int `toml:"port"`
}

func loadConfig(path string) (Config, error) {
	cfg := Config{
		LLM:    LLMConfig{BaseURL: "http://localhost:1234", Model: "local-model", Temperature: 0.0},
		Server: ServerConfig{Port: 8080},
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// ---- API types -------------------------------------------------------------

type MoveRequest struct {
	Board      [8][8]int `json:"board"`
	ValidMoves []Move    `json:"validMoves"`
}

type Move struct {
	R int `json:"r"`
	C int `json:"c"`
}

type MoveResponse struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// ---- LLM client ------------------------------------------------------------

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var jsonRe = regexp.MustCompile(`\{[^{}]*\}`)

func askLLM(cfg LLMConfig, req MoveRequest) (MoveResponse, error) {
	prompt := buildPrompt(req.Board, req.ValidMoves)

	body, err := json.Marshal(chatRequest{
		Model: cfg.Model,
		Messages: []message{
			{Role: "system", Content: "You are an expert Othello (Reversi) AI playing as White. Reason briefly about your best move, then output a single JSON object on the last line."},
			{Role: "user", Content: prompt},
		},
		Temperature: cfg.Temperature,
		MaxTokens:   256,
	})
	if err != nil {
		return MoveResponse{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, cfg.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return MoveResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return MoveResponse{}, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return MoveResponse{}, err
	}

	var chat chatResponse
	if err := json.Unmarshal(raw, &chat); err != nil || len(chat.Choices) == 0 {
		return MoveResponse{}, fmt.Errorf("unexpected LLM response: %s", raw)
	}

	content := chat.Choices[0].Message.Content
	log.Printf("LLM response: %s", content)

	var move MoveResponse
	if err := json.Unmarshal([]byte(content), &move); err != nil {
		// Try to extract JSON from surrounding text
		if m := jsonRe.FindString(content); m != "" {
			if err2 := json.Unmarshal([]byte(m), &move); err2 != nil {
				return MoveResponse{}, fmt.Errorf("cannot parse LLM move from: %s", content)
			}
		} else {
			return MoveResponse{}, fmt.Errorf("cannot parse LLM move from: %s", content)
		}
	}
	return move, nil
}

// squareWeights is the classic Othello positional weight table.
var squareWeights = [8][8]int{
	{100, -20, 10, 5, 5, 10, -20, 100},
	{-20, -40, -5, -5, -5, -5, -40, -20},
	{10, -5, 3, 3, 3, 3, -5, 10},
	{5, -5, 3, 3, 3, 3, -5, 5},
	{5, -5, 3, 3, 3, 3, -5, 5},
	{10, -5, 3, 3, 3, 3, -5, 10},
	{-20, -40, -5, -5, -5, -5, -40, -20},
	{100, -20, 10, 5, 5, 10, -20, 100},
}

func buildPrompt(board [8][8]int, validMoves []Move) string {
	var sb strings.Builder

	sb.WriteString(`## Your role
You are an expert Othello (Reversi) player. You play as White (W). Your goal is to win.

## Strategic priorities (follow in order)
1. TAKE corners immediately: (0,0) (0,7) (7,0) (7,7) — stable forever, highest value
2. AVOID X-squares: (1,1) (1,6) (6,1) (6,6) — directly gives opponent corner access
3. AVOID C-squares next to unowned corners: (0,1) (1,0) (0,6) (1,7) (6,0) (7,1) (6,7) (7,6)
4. Prefer moves that leave the opponent with fewer valid moves (mobility advantage)
5. Among remaining options, prefer higher square value

## Square value map (higher = better)
     col: 0    1    2    3    4    5    6    7
row 0:  100  -20   10    5    5   10  -20  100
row 1:  -20  -40   -5   -5   -5   -5  -40  -20
row 2:   10   -5    3    3    3    3   -5   10
row 3:    5   -5    3    3    3    3   -5    5
row 4:    5   -5    3    3    3    3   -5    5
row 5:   10   -5    3    3    3    3   -5   10
row 6:  -20  -40   -5   -5   -5   -5  -40  -20
row 7:  100  -20   10    5    5   10  -20  100

## Current board (B=Black/human, W=White/you, .=empty)
`)

	sb.WriteString("     col: 0 1 2 3 4 5 6 7\n")
	for r := 0; r < 8; r++ {
		fmt.Fprintf(&sb, "row %d:    ", r)
		for c := 0; c < 8; c++ {
			switch board[r][c] {
			case 1:
				sb.WriteString("B ")
			case 2:
				sb.WriteString("W ")
			default:
				sb.WriteString(". ")
			}
		}
		sb.WriteByte('\n')
	}

	sb.WriteString("\n## Your valid moves (row, col) → square value\n")
	for _, m := range validMoves {
		w := squareWeights[m.R][m.C]
		fmt.Fprintf(&sb, "- (%d, %d) → %+d\n", m.R, m.C, w)
	}

	sb.WriteString("\nBriefly explain your choice (1–2 sentences), then output your move as JSON on the final line:\n{\"row\": <0-7>, \"col\": <0-7>}")
	return sb.String()
}

// ---- HTTP handlers ---------------------------------------------------------

func makeMoveHandler(cfg LLMConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req MoveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.ValidMoves) == 0 {
			http.Error(w, "no valid moves", http.StatusBadRequest)
			return
		}

		move, err := askLLM(cfg, req)
		if err != nil {
			log.Printf("LLM error: %v — falling back to random move", err)
			random := req.ValidMoves[rand.Intn(len(req.ValidMoves))]
			move = MoveResponse{Row: random.R, Col: random.C}
		}

		// Validate returned move is in validMoves
		valid := false
		for _, m := range req.ValidMoves {
			if m.R == move.Row && m.C == move.Col {
				valid = true
				break
			}
		}
		if !valid {
			log.Printf("LLM returned invalid move (%d,%d) — falling back to random", move.Row, move.Col)
			random := req.ValidMoves[rand.Intn(len(req.ValidMoves))]
			move = MoveResponse{Row: random.R, Col: random.C}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(move)
	}
}

// ---- Main ------------------------------------------------------------------

func main() {
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	http.Handle("/", http.FileServer(http.FS(static)))
	http.HandleFunc("/api/move", makeMoveHandler(cfg.LLM))

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("llm-othello listening on http://localhost%s (LLM: %s, model: %s)", addr, cfg.LLM.BaseURL, cfg.LLM.Model)
	log.Fatal(http.ListenAndServe(addr, nil))
}
