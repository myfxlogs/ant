package agent

import (
	"strings"
	"sync"
	"text/template"
)

// promptData holds all user-controlled inputs for prompt templates.
// User-supplied content is wrapped in XML tags to prevent prompt injection.
type promptData struct {
	Message         string // user NL strategy description
	Feedback        string // user plan feedback
	MQLSource       string // MQL source code (bridge)
	PrevCode        string // previous LLM output (retry)
	PrevPython      string // previous Python output (bridge retry)
	ErrorMsg        string // compile/backtest error
	Params          string // formatted params block
	ProfileBlock    string // formatted profile block
	MemoryBlock     string // formatted session memory block
	CoverageBlock   string // formatted coverage report
	PlanBlock       string // formatted plan block
	BacktestBlock   string // formatted backtest results
	TradesBlock     string // formatted trades
	RetrospectBlock string // formatted retrospect summary
}

// wrapXML wraps user-controlled content in XML-style tags for prompt injection protection.
// This follows the pattern recommended by Anthropic and OpenAI for isolating untrusted input.
func wrapXML(tag, content string) string {
	return "<" + tag + ">\n" + content + "\n</" + tag + ">"
}

// sanitizeInput removes any existing XML-like tags from user input to prevent tag spoofing.
// This is a defense-in-depth measure — the primary protection is wrapping in our own tags.
func sanitizeInput(s string) string {
	// Strip angle brackets that could form fake closing tags
	s = strings.ReplaceAll(s, "</", "< /")
	return s
}

// promptRenderer manages parsed templates for reuse.
type promptRenderer struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
}

var renderer = &promptRenderer{templates: make(map[string]*template.Template)}

// renderPrompt parses (once) and executes a text/template, returning the rendered string.
// Templates use {{.Field}} for structured data and {{.UserInput}} for sanitized user content.
func renderPrompt(name, tmplText string, data interface{}) (string, error) {
	renderer.mu.RLock()
	t, ok := renderer.templates[name]
	renderer.mu.RUnlock()
	if !ok {
		var err error
		t, err = template.New(name).Parse(tmplText)
		if err != nil {
			return "", err
		}
		renderer.mu.Lock()
		renderer.templates[name] = t
		renderer.mu.Unlock()
	}
	var sb strings.Builder
	if err := t.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}
