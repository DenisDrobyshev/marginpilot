// Package domain holds the gateway's core types. It has no knowledge of HTTP,
// JSON wire formats beyond struct tags, providers, or databases.
package domain

// ChatRequest is a minimal OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message is a single chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is a minimal OpenAI-compatible chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is one completion candidate.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage reports token consumption for a call — the raw material for metering.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
