// Package provider contains outbound adapters implementing port.LLMProvider.
package provider

import (
	"context"
	"strings"

	"github.com/marginpilot/gateway/internal/domain"
)

// Echo is a stand-in provider for local dev and tests. It echoes the last user
// message and reports token counts with a naive whitespace heuristic, so the
// metering path sees realistic data before a real OpenAI/Anthropic client and
// a proper tokenizer are wired in.
type Echo struct{}

// NewEcho constructs the echo provider.
func NewEcho() *Echo { return &Echo{} }

// Name identifies the provider in usage events.
func (e *Echo) Name() string { return "echo" }

// Complete returns a canned completion echoing the last message.
func (e *Echo) Complete(_ context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	var prompt strings.Builder
	last := ""
	for _, m := range req.Messages {
		prompt.WriteString(m.Content)
		prompt.WriteByte(' ')
		last = m.Content
	}

	reply := "echo: " + last
	in := countTokens(prompt.String())
	out := countTokens(reply)

	model := req.Model
	if model == "" {
		model = "echo-1"
	}

	return domain.ChatResponse{
		ID:     "chatcmpl-echo",
		Object: "chat.completion",
		Model:  model,
		Choices: []domain.Choice{{
			Index:        0,
			Message:      domain.Message{Role: "assistant", Content: reply},
			FinishReason: "stop",
		}},
		Usage: domain.Usage{PromptTokens: in, CompletionTokens: out, TotalTokens: in + out},
	}, nil
}

func countTokens(s string) int {
	if n := len(strings.Fields(s)); n > 0 {
		return n
	}
	return 1
}
