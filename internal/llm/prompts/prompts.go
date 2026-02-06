package prompts

import (
	_ "embed"
	"os"
	"strings"
	"text/template"
)

//go:embed system.md
var defaultSystemPrompt string

//go:embed decision.md
var defaultDecisionPrompt string

type DecisionData struct {
	Context     string
	Timestamp   string
	Close       float64
	SMA         float64
	PositionQty int
	MaxQty      int
}

func DefaultSystemPrompt() string {
	return defaultSystemPrompt
}

func DefaultDecisionPrompt() string {
	return defaultDecisionPrompt
}

func LoadTemplate(path string, fallback string) string {
	if path == "" {
		return fallback
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(contents)
}

func RenderDecisionPrompt(templateText string, data DecisionData) (string, error) {
	tmpl, err := template.New("decision").Parse(templateText)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}
