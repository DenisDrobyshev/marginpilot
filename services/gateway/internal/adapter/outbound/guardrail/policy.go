package guardrail

import (
	"regexp"
	"strings"

	"github.com/marginpilot/gateway/internal/app"
	"github.com/marginpilot/gateway/internal/domain"
)

// Mode selects how the policy reacts to detected PII.
type Mode string

const (
	// ModeRedact replaces PII with a placeholder and lets the request through.
	ModeRedact Mode = "redact"
	// ModeBlock rejects any request containing PII.
	ModeBlock Mode = "block"
)

// piiPatterns are deliberately conservative examples (email, card-like, phone).
var piiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
	regexp.MustCompile(`\b(?:\d[ -]?){13,16}\b`),
	regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`),
}

// Policy detects PII (redacts or blocks per Mode) and blocks denylisted terms.
type Policy struct {
	mode     Mode
	denylist []string
}

// NewPolicy builds a policy. denylist terms are matched case-insensitively.
func NewPolicy(mode Mode, denylist []string) *Policy {
	var dl []string
	for _, d := range denylist {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			dl = append(dl, d)
		}
	}
	return &Policy{mode: mode, denylist: dl}
}

// Check inspects each message, blocking on denylist hits and redacting or
// blocking on PII depending on the mode.
func (p *Policy) Check(req domain.ChatRequest) (domain.ChatRequest, error) {
	for i, m := range req.Messages {
		lower := strings.ToLower(m.Content)
		for _, term := range p.denylist {
			if strings.Contains(lower, term) {
				return domain.ChatRequest{}, app.ErrBlocked
			}
		}

		redacted, found := m.Content, false
		for _, re := range piiPatterns {
			if re.MatchString(redacted) {
				found = true
				redacted = re.ReplaceAllString(redacted, "[REDACTED]")
			}
		}
		if found {
			if p.mode == ModeBlock {
				return domain.ChatRequest{}, app.ErrBlocked
			}
			req.Messages[i].Content = redacted
		}
	}
	return req, nil
}
