package llm

import (
	"context"
	"testing"
	"time"
)

type mockProvider struct {
	name   string
	models []Model
}

func (m *mockProvider) Name() string    { return m.name }
func (m *mockProvider) Models() []Model { return m.models }

func (m *mockProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	return &Response{
		Content:    "mock response",
		Model:      "mock-model",
		TokensUsed: 10,
		Duration:   time.Millisecond * 100,
		CostUSD:    0.0,
	}, nil
}

func (m *mockProvider) Embed(ctx context.Context, texts []string) ([]Embedding, error) {
	vec := make([]float32, 128)
	for i := range vec {
		vec[i] = 0.5
	}
	embeddings := make([]Embedding, len(texts))
	for i := range texts {
		embeddings[i] = Embedding{Vector: vec, Model: "mock-embed"}
	}
	return embeddings, nil
}

func TestRouter_Register(t *testing.T) {
	router := NewRouter()

	provider := &mockProvider{name: "test", models: []Model{{Name: "test-model"}}}
	router.Register(provider)

	p, ok := router.GetProvider("test")
	if !ok {
		t.Error("Expected provider to be registered")
	}
	if p.Name() != "test" {
		t.Errorf("Expected provider name 'test', got '%s'", p.Name())
	}
}

func TestRouter_GetProvider_NotFound(t *testing.T) {
	router := NewRouter()

	_, ok := router.GetProvider("nonexistent")
	if ok {
		t.Error("Expected provider to not be found")
	}
}

func TestRouter_Default(t *testing.T) {
	router := NewRouter()

	provider1 := &mockProvider{name: "provider1", models: []Model{}}
	provider2 := &mockProvider{name: "provider2", models: []Model{}}
	router.Register(provider1)
	router.Register(provider2)

	p, ok := router.Default()
	if !ok {
		t.Error("Expected default provider to be set")
	}
	if p.Name() != "provider1" {
		t.Errorf("Expected first registered provider as default, got '%s'", p.Name())
	}
}

func TestRouter_Default_NoProviders(t *testing.T) {
	router := NewRouter()

	_, ok := router.Default()
	if ok {
		t.Error("Expected no default provider when none registered")
	}
}

func TestRouter_Chat_UsesDefaultProvider(t *testing.T) {
	router := NewRouter()

	provider := &mockProvider{
		name:   "default",
		models: []Model{{Name: "default-model"}},
	}
	router.Register(provider)

	messages := []Message{{Role: "user", Content: "hello"}}
	resp, err := router.Chat(context.Background(), messages, ChatOptions{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp.Content != "mock response" {
		t.Errorf("Expected mock response, got '%s'", resp.Content)
	}
}

func TestRouter_Chat_SpecificProvider(t *testing.T) {
	router := NewRouter()

	provider1 := &mockProvider{name: "provider1", models: []Model{{Name: "model1"}}}
	provider2 := &mockProvider{name: "provider2", models: []Model{{Name: "model2"}}}
	router.Register(provider1)
	router.Register(provider2)

	messages := []Message{{Role: "user", Content: "hello"}}
	opts := ChatOptions{Model: "provider2"}
	resp, err := router.Chat(context.Background(), messages, opts)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp.Model != "mock-model" {
		t.Errorf("Expected mock model, got '%s'", resp.Model)
	}
}

func TestNewOllamaProvider(t *testing.T) {
	provider := NewOllamaProvider("")
	if provider == nil {
		t.Error("NewOllamaProvider should not return nil")
	}
	if provider.Name() != "ollama" {
		t.Errorf("Expected name 'ollama', got '%s'", provider.Name())
	}
	if len(provider.Models()) == 0 {
		t.Error("Expected at least one default model")
	}
}

func TestNewOllamaProvider_CustomURL(t *testing.T) {
	provider := NewOllamaProvider("http://custom:11434")
	if provider == nil {
		t.Error("NewOllamaProvider should not return nil")
	}
}

func TestOllamaProvider_DefaultModels(t *testing.T) {
	provider := NewOllamaProvider("")

	models := provider.Models()
	if len(models) == 0 {
		t.Error("Expected at least one default model")
	}

	foundQwen := false
	for _, m := range models {
		if m.Name == "qwen2.5-coder:32b" {
			foundQwen = true
			break
		}
	}
	if !foundQwen {
		t.Error("Expected qwen2.5-coder:32b in default models")
	}
}

func TestOllamaProvider_IsAvailable(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:9999")

	available := provider.IsAvailable(context.Background())
	if available {
		t.Error("Expected false for unavailable server")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider("test-key")
	if provider.Name() != "openai" {
		t.Errorf("Expected name 'openai', got '%s'", provider.Name())
	}
}

func TestOpenAIProvider_Models(t *testing.T) {
	provider := NewOpenAIProvider("test-key")
	models := provider.Models()

	if len(models) == 0 {
		t.Error("Expected at least one model")
	}

	for _, m := range models {
		if m.Provider != "openai" {
			t.Errorf("Expected provider 'openai', got '%s'", m.Provider)
		}
		if m.CostPer1KTokens <= 0 {
			t.Errorf("Expected cost > 0 for OpenAI models, got %f", m.CostPer1KTokens)
		}
	}
}

func TestModel_CostPer1KTokens(t *testing.T) {
	model := Model{
		Name:            "test",
		CostPer1KTokens: 0.001,
	}

	if model.CostPer1KTokens != 0.001 {
		t.Errorf("Expected CostPer1KTokens 0.001, got %f", model.CostPer1KTokens)
	}
}

func TestEmbedding_Model(t *testing.T) {
	embedding := Embedding{
		Vector: []float32{0.1, 0.2, 0.3},
		Model:  "test-embed",
	}

	if embedding.Model != "test-embed" {
		t.Errorf("Expected model 'test-embed', got '%s'", embedding.Model)
	}
	if len(embedding.Vector) != 3 {
		t.Errorf("Expected 3 vector elements, got %d", len(embedding.Vector))
	}
}

func TestResponse_Duration(t *testing.T) {
	resp := &Response{
		Content:    "test",
		Model:      "test",
		TokensUsed: 10,
		Duration:   time.Second * 5,
		CostUSD:    0.01,
	}

	if resp.Duration != time.Second*5 {
		t.Errorf("Expected 5 seconds, got %v", resp.Duration)
	}
}

func TestResponse_CostUSD(t *testing.T) {
	resp := &Response{
		Content:    "test",
		Model:      "test",
		TokensUsed: 1000,
		CostUSD:    0.05,
	}

	if resp.CostUSD != 0.05 {
		t.Errorf("Expected 0.05, got %f", resp.CostUSD)
	}
}

func TestChatOptions_Defaults(t *testing.T) {
	opts := ChatOptions{}

	if opts.Model != "" {
		t.Errorf("Expected empty model, got '%s'", opts.Model)
	}
	if opts.MaxTokens != 0 {
		t.Errorf("Expected 0 max tokens, got %d", opts.MaxTokens)
	}
	if opts.Temperature != 0 {
		t.Errorf("Expected 0 temperature, got %f", opts.Temperature)
	}
	if opts.SystemPrompt != "" {
		t.Errorf("Expected empty system prompt, got '%s'", opts.SystemPrompt)
	}
}

func TestChatOptions_CustomValues(t *testing.T) {
	opts := ChatOptions{
		Model:        "gpt-4o",
		MaxTokens:    1000,
		Temperature:  0.7,
		SystemPrompt: "You are a code reviewer",
	}

	if opts.Model != "gpt-4o" {
		t.Errorf("Expected 'gpt-4o', got '%s'", opts.Model)
	}
	if opts.MaxTokens != 1000 {
		t.Errorf("Expected 1000, got %d", opts.MaxTokens)
	}
	if opts.Temperature != 0.7 {
		t.Errorf("Expected 0.7, got %f", opts.Temperature)
	}
	if opts.SystemPrompt != "You are a code reviewer" {
		t.Errorf("Expected system prompt, got '%s'", opts.SystemPrompt)
	}
}

func TestMessage_Role(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	if msg.Role != "user" {
		t.Errorf("Expected 'user', got '%s'", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", msg.Content)
	}
}

func TestRouter_Chat_NoProviders(t *testing.T) {
	router := NewRouter()

	_, err := router.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, ChatOptions{})
	if err == nil {
		t.Error("Expected error when no providers registered")
	}
}

func TestOllamaProvider_Embed(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:9999")

	ctx := context.Background()
	embeddings, err := provider.Embed(ctx, []string{"hello world"})

	if err == nil {
		t.Error("Expected error for unavailable server")
	}
	if embeddings != nil {
		t.Error("Expected nil embeddings for unavailable server")
	}
}
