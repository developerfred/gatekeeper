package llm

import (
	"context"
	"fmt"
	"time"
)

type Message struct {
	Role    string
	Content string
}

type Response struct {
	Content    string
	Model      string
	TokensUsed int
	Duration   time.Duration
	CostUSD    float64
}

type Provider interface {
	Chat(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error)
	Models() []Model
	Embed(ctx context.Context, texts []string) ([]Embedding, error)
	Name() string
}

type ChatOptions struct {
	Model        string
	MaxTokens    int
	Temperature  float64
	SystemPrompt string
}

type Model struct {
	Name            string
	Provider        string
	ContextLen      int
	CostPer1KTokens float64
	IsFast          bool
}

type Embedding struct {
	Vector []float32
	Model  string
}

type Router struct {
	providers       map[string]Provider
	defaultProvider string
}

func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
	}
}

func (r *Router) Register(provider Provider) {
	r.providers[provider.Name()] = provider
	if r.defaultProvider == "" {
		r.defaultProvider = provider.Name()
	}
}

func (r *Router) GetProvider(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

func (r *Router) Default() (Provider, bool) {
	return r.GetProvider(r.defaultProvider)
}

func (r *Router) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	if len(r.providers) == 0 {
		return nil, fmt.Errorf("no LLM providers registered")
	}

	providerName := opts.Model
	if providerName == "" {
		providerName = r.defaultProvider
	}

	provider, ok := r.providers[providerName]
	if !ok {
		for _, p := range r.providers {
			provider = p
			break
		}
	}

	return provider.Chat(ctx, messages, opts)
}
