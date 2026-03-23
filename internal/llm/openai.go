package llm

import (
	"context"
	"time"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	models []Model
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	config := openai.DefaultConfig(apiKey)
	client := openai.NewClientWithConfig(config)

	models := []Model{
		{Name: "gpt-4o", Provider: "openai", ContextLen: 128000, CostPer1KTokens: 0.005, IsFast: false},
		{Name: "gpt-4o-mini", Provider: "openai", ContextLen: 128000, CostPer1KTokens: 0.00015, IsFast: true},
		{Name: "o3-mini", Provider: "openai", ContextLen: 65536, CostPer1KTokens: 0.001, IsFast: true},
		{Name: "gpt-4-turbo", Provider: "openai", ContextLen: 128000, CostPer1KTokens: 0.01, IsFast: false},
	}

	return &OpenAIProvider{
		client: client,
		models: models,
	}
}

func NewOpenAIProviderWithClient(client *openai.Client) *OpenAIProvider {
	models := []Model{
		{Name: "gpt-4o", Provider: "openai", ContextLen: 128000, CostPer1KTokens: 0.005, IsFast: false},
		{Name: "gpt-4o-mini", Provider: "openai", ContextLen: 128000, CostPer1KTokens: 0.00015, IsFast: true},
		{Name: "o3-mini", Provider: "openai", ContextLen: 65536, CostPer1KTokens: 0.001, IsFast: true},
	}

	return &OpenAIProvider{
		client: client,
		models: models,
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Models() []Model { return p.models }

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	model := opts.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	reqMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		reqMessages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	req := openai.ChatCompletionRequest{
		Model: model,
	}

	if opts.MaxTokens > 0 {
		req.MaxTokens = opts.MaxTokens
	}

	if opts.Temperature > 0 {
		req.Temperature = float32(opts.Temperature)
	}

	if opts.SystemPrompt != "" {
		reqMessages = append([]openai.ChatCompletionMessage{
			{Role: "system", Content: opts.SystemPrompt},
		}, reqMessages...)
	}

	req.Messages = reqMessages

	start := time.Now()
	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	duration := time.Since(start)

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	tokens := resp.Usage.TotalTokens
	cost := calculateCost(tokens, model)

	return &Response{
		Content:    content,
		Model:      model,
		TokensUsed: tokens,
		Duration:   duration,
		CostUSD:    cost,
	}, nil
}

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([]Embedding, error) {
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: "text-embedding-3-small",
	}

	resp, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}

	embeddings := make([]Embedding, len(texts))
	for i, data := range resp.Data {
		embeddings[i] = Embedding{
			Vector: data.Embedding,
			Model:  "text-embedding-3-small",
		}
	}

	return embeddings, nil
}

func calculateCost(tokens int, model string) float64 {
	costPer1K := map[string]float64{
		"gpt-4o":      0.005,
		"gpt-4o-mini": 0.00015,
		"o3-mini":     0.001,
		"gpt-4-turbo": 0.01,
	}

	if c, ok := costPer1K[model]; ok {
		return float64(tokens) / 1000.0 * c
	}
	return 0.0
}
