package aiops

import "context"

// Prompt is the bounded context sent to an AI text generator.
type Prompt struct {
	Kind   InsightKind
	System string
	User   string
}

// GeneratedContent is a model or fallback text response.
type GeneratedContent struct {
	Content  string
	Provider string
	Model    string
}

// Generator produces read-only operational text from bounded context.
type Generator interface {
	Generate(ctx context.Context, prompt Prompt) (GeneratedContent, error)
}
