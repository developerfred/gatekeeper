package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	baseURL string
	client  *http.Client
	models  []Model
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Model         string `json:"model"`
	TotalDuration int64  `json:"total_duration"`
	EvalCount     int    `json:"eval_count"`
}

type ollamaListResponse struct {
	Models []struct {
		Name       string `json:"name"`
		ModifiedAt string `json:"modified_at"`
	} `json:"models"`
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	return &OllamaProvider{
		baseURL: baseURL,
		client:  client,
		models: []Model{
			{Name: "qwen2.5-coder:32b", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: false},
			{Name: "qwen2.5-coder:7b", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: true},
			{Name: "deepseek-coder:33b", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: false},
			{Name: "codellama:34b", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: false},
			{Name: "llama3.1:8b", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: true},
			{Name: "qwen3.5", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: false},
			{Name: "phi4:latest", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: true},
			{Name: "mistral:7b", Provider: "ollama", ContextLen: 32768, CostPer1KTokens: 0.0, IsFast: true},
		},
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) Models() []Model {
	models, err := p.listModels()
	if err != nil {
		return p.models
	}

	if len(models) > 0 {
		result := make([]Model, 0, len(models))
		for _, m := range models {
			result = append(result, Model{
				Name:            m.Name,
				Provider:        "ollama",
				ContextLen:      32768,
				CostPer1KTokens: 0.0,
				IsFast:          false,
			})
		}
		return result
	}

	return p.models
}

type ollamaModel struct {
	Name string `json:"name"`
}

func (p *OllamaProvider) listModels() ([]ollamaModel, error) {
	req, err := http.NewRequest(http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var listResp ollamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	models := make([]ollamaModel, len(listResp.Models))
	for i, m := range listResp.Models {
		models[i] = ollamaModel{Name: m.Name}
	}

	return models, nil
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	model := opts.Model
	if model == "" {
		model = "qwen2.5-coder:32b"
	}

	reqMessages := make([]Message, len(messages))
	copy(reqMessages, messages)

	ollamaReq := ollamaChatRequest{
		Model:    model,
		Messages: reqMessages,
		Stream:   false,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}

	duration := time.Since(start)

	tokens := ollamaResp.EvalCount
	if tokens == 0 {
		tokens = len(ollamaResp.Message.Content) / 4
	}

	return &Response{
		Content:    ollamaResp.Message.Content,
		Model:      model,
		TokensUsed: tokens,
		Duration:   duration,
		CostUSD:    0.0,
	}, nil
}

func (p *OllamaProvider) Embed(ctx context.Context, texts []string) ([]Embedding, error) {
	model := "nomic-embed-text"
	ollamaReq := ollamaEmbedRequest{
		Model: model,
		Input: texts,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embeddings returned status %d", resp.StatusCode)
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, err
	}

	embeddings := make([]Embedding, len(embedResp.Embeddings))
	for i, vec := range embedResp.Embeddings {
		embeddings[i] = Embedding{
			Vector: vec,
			Model:  model,
		}
	}

	return embeddings, nil
}

func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
