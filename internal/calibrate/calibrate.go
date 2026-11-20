package calibrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"

	"github.com/lucawalz/strata/internal/note"
)

const (
	countTokensPath  = "/v1/messages/count_tokens"
	anthropicVersion = "2023-06-01"
	roundTo          = 100
)

var capTokens = [3]int{300, 1500, 3000}

type Result struct {
	Notes, Chars, Tokens int
	CharsPerToken        float64
}

func (r Result) ProposedCaps() [3]int {
	var caps [3]int
	for i, tok := range capTokens {
		caps[i] = int(math.Round(float64(tok)*r.CharsPerToken/roundTo)) * roundTo
	}
	return caps
}

func Run(ctx context.Context, client *http.Client, baseURL, apiKey, model string, files []string) (Result, error) {
	var res Result
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		n, err := note.Parse(path, src)
		if err != nil {
			return Result{}, err
		}
		tokens, err := countTokens(ctx, client, baseURL, apiKey, model, string(n.Body))
		if err != nil {
			return Result{}, fmt.Errorf("%s: %w", path, err)
		}
		res.Notes++
		res.Chars += n.Chars
		res.Tokens += tokens
	}
	if res.Tokens == 0 {
		return Result{}, errors.New("no tokens counted")
	}
	res.CharsPerToken = float64(res.Chars) / float64(res.Tokens)
	return res, nil
}

func countTokens(ctx context.Context, client *http.Client, baseURL, apiKey, model, body string) (int, error) {
	payload, err := json.Marshal(map[string]any{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": body}},
	})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+countTokensPath, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("count_tokens returned status %d", resp.StatusCode)
	}
	var out struct {
		InputTokens int `json:"input_tokens"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return 0, err
	}
	return out.InputTokens, nil
}
