// Package rules is Trustwright's rule format: what must hold on the map.
//
// A rule is data, not code. Each one names a check operation and the nodes and
// edges it applies to, so the Go checker can decide it the same way every time.
// Rules imported from a project's own docs start as drafts and only count once the
// user makes them active.
package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/altrudev/trustwright/model"
)

// Format is the value of RuleSet.Format this package reads and writes.
const Format = "trustwright.rules/1"

// RuleSet is one rules file.
type RuleSet struct {
	Format string `json:"format"`
	Rules  []Rule `json:"rules"`
}

// Rule statuses.
const (
	StatusDraft  = "draft"  // imported or proposed; reported, never blocks
	StatusActive = "active" // confirmed by the user; a break is a finding
	StatusOff    = "off"    // kept for history, not checked
)

// Severities.
const (
	SeverityHigh   = "high"
	SeverityMedium = "medium"
	SeverityLow    = "low"
)

// Rule is one thing that must hold.
type Rule struct {
	ID       string            `json:"id"`              // lower-case, dot-separated: kernel.frames-authenticated
	Title    string            `json:"title"`           // short, plain words
	Claim    string            `json:"claim,omitempty"` // the rule as a sentence a person can check
	Status   string            `json:"status"`
	Severity string            `json:"severity"`
	Family   string            `json:"family,omitempty"` // template family A-G that can fix it
	Src      []model.SourceRef `json:"src,omitempty"`    // where the rule is written down
	Check    Check             `json:"check"`
}

// Check operations.
const (
	// OpEachHasEdge: every Subject node has at least one Edge to (or, with
	// Direction "in", from) a node matching Object.
	OpEachHasEdge = "each-has-edge"
	// OpNoEdge: no Subject node has an Edge to (or from) a node matching Object.
	OpNoEdge = "no-edge"
	// OpEachAttr: every Subject node has every attribute in Want, with that value
	// ("*" means any value).
	OpEachAttr = "each-attr"
	// OpAllPathsVia: every path from a Subject node to an Object node, following
	// edges of kind Edge (any kind when empty), passes through a Via node.
	OpAllPathsVia = "all-paths-via"
	// OpSameSet: the values of attribute Key on Subject nodes equal those on
	// Object nodes. Used for "the spec says X, the code does Y".
	OpSameSet = "same-set"
)

// Check is the machine form of a rule.
type Check struct {
	Op        string            `json:"op"`
	Subject   Selector          `json:"subject"`
	Edge      string            `json:"edge,omitempty"`
	Direction string            `json:"direction,omitempty"` // out (default) or in
	Object    *Selector         `json:"object,omitempty"`
	Via       *Selector         `json:"via,omitempty"`
	Want      map[string]string `json:"want,omitempty"`
	Key       string            `json:"key,omitempty"`
}

// Selector picks nodes. Every field that is set must match; an empty selector
// matches every node. ID may use path.Match patterns, such as "worker:*". An
// attribute value of "*" matches any value, as long as the attribute is there.
type Selector struct {
	ID    string            `json:"id,omitempty"`
	Kind  string            `json:"kind,omitempty"`
	Type  string            `json:"type,omitempty"`
	Group string            `json:"group,omitempty"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

// Empty reports whether the selector would match every node.
func (s Selector) Empty() bool {
	return s.ID == "" && s.Kind == "" && s.Type == "" && s.Group == "" && len(s.Attrs) == 0
}

// Match reports whether the node is picked by the selector.
func (s Selector) Match(n model.Node) bool {
	if s.ID != "" {
		if ok, err := path.Match(s.ID, n.ID); err != nil || !ok {
			return false
		}
	}
	if s.Kind != "" && s.Kind != n.Kind {
		return false
	}
	if s.Type != "" && s.Type != n.Type {
		return false
	}
	if s.Group != "" && s.Group != n.Group {
		return false
	}
	for k, v := range s.Attrs {
		got, ok := n.Attrs[k]
		if !ok || (v != "*" && got != v) {
			return false
		}
	}
	return true
}

// Select returns the ids of the graph's nodes the selector picks, in graph order.
func (s Selector) Select(g *model.Graph) []string {
	var out []string
	for _, n := range g.Nodes {
		if s.Match(n) {
			out = append(out, n.ID)
		}
	}
	return out
}

func (s Selector) validate() error {
	if s.ID != "" {
		if _, err := path.Match(s.ID, ""); err != nil {
			return fmt.Errorf("bad id pattern %q", s.ID)
		}
	}
	return nil
}

var idRe = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*(?:\.[a-z0-9]+(?:[-_][a-z0-9]+)*)*$`)

// Decode reads and validates a rules file. Unknown keys are an error.
func Decode(r io.Reader) (*RuleSet, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var rs RuleSet
	if err := dec.Decode(&rs); err != nil {
		return nil, fmt.Errorf("decode rules: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("decode rules: extra data after the rule set")
	}
	if err := rs.Validate(); err != nil {
		return nil, err
	}
	return &rs, nil
}

// Validate checks every rule. It reports every problem, not just the first.
func (rs *RuleSet) Validate() error {
	var errs []string
	if rs.Format != Format {
		errs = append(errs, fmt.Sprintf("format %q: want %q", rs.Format, Format))
	}
	seen := map[string]bool{}
	for i, r := range rs.Rules {
		name := r.ID
		if name == "" {
			name = fmt.Sprintf("#%d", i)
		}
		if seen[r.ID] {
			errs = append(errs, fmt.Sprintf("rule %s: duplicate id", name))
		}
		seen[r.ID] = true
		for _, e := range r.problems() {
			errs = append(errs, fmt.Sprintf("rule %s: %s", name, e))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid rules:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// Validate checks one rule on its own.
func (r Rule) Validate() error {
	if p := r.problems(); len(p) > 0 {
		return fmt.Errorf("rule %s: %s", r.ID, strings.Join(p, "; "))
	}
	return nil
}

func (r Rule) problems() []string {
	var p []string
	add := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }

	if !idRe.MatchString(r.ID) {
		add("id %q: use lower-case words joined by - and .", r.ID)
	}
	if strings.TrimSpace(r.Title) == "" {
		add("empty title")
	}
	if !oneOf(r.Status, StatusDraft, StatusActive, StatusOff) {
		add("status %q: want draft, active or off", r.Status)
	}
	if !oneOf(r.Severity, SeverityHigh, SeverityMedium, SeverityLow) {
		add("severity %q: want high, medium or low", r.Severity)
	}
	if r.Family != "" && (len(r.Family) != 1 || r.Family[0] < 'A' || r.Family[0] > 'G') {
		add("family %q: want a letter A-G", r.Family)
	}
	for _, s := range r.Src {
		if err := s.Validate(); err != nil {
			add("%v", err)
		}
	}

	c := r.Check
	if c.Subject.Empty() {
		add("check has no subject: it would apply to every node")
	}
	for _, s := range []*Selector{&c.Subject, c.Object, c.Via} {
		if s != nil {
			if err := s.validate(); err != nil {
				add("%v", err)
			}
		}
	}
	if c.Direction != "" && !oneOf(c.Direction, "out", "in") {
		add("direction %q: want out or in", c.Direction)
	}

	// Which fields each op needs (+) and must not have (-).
	need := func(ok bool, what string) {
		if !ok {
			add("%s needs %s", c.Op, what)
		}
	}
	deny := func(set bool, what string) {
		if set {
			add("%s does not use %s", c.Op, what)
		}
	}
	switch c.Op {
	case OpEachHasEdge, OpNoEdge:
		need(c.Edge != "", "edge")
		need(c.Object != nil, "object")
		deny(c.Via != nil, "via")
		deny(len(c.Want) > 0, "want")
		deny(c.Key != "", "key")
	case OpEachAttr:
		need(len(c.Want) > 0, "want")
		deny(c.Edge != "", "edge")
		deny(c.Object != nil, "object")
		deny(c.Via != nil, "via")
		deny(c.Key != "", "key")
		deny(c.Direction != "", "direction")
	case OpAllPathsVia:
		need(c.Object != nil, "object")
		need(c.Via != nil && !c.Via.Empty(), "via")
		deny(len(c.Want) > 0, "want")
		deny(c.Key != "", "key")
		deny(c.Direction != "", "direction")
	case OpSameSet:
		need(c.Object != nil, "object")
		need(c.Key != "", "key")
		deny(c.Edge != "", "edge")
		deny(c.Via != nil, "via")
		deny(len(c.Want) > 0, "want")
		deny(c.Direction != "", "direction")
	default:
		add("unknown op %q", c.Op)
	}
	return p
}

func oneOf(s string, set ...string) bool {
	for _, x := range set {
		if s == x {
			return true
		}
	}
	return false
}
