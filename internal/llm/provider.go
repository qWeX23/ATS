package llm

import "context"

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	SupportsTools() bool
}
