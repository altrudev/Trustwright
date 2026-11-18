// Package model is Trustwright's map of a system: what exists (nodes), how the
// parts connect (edges), what the user can switch (levers) and what follows when
// they do (effects).
//
// The JSON format extends Ripple's engine model export (engine "ripple/0.1"), so a
// model copied out of Ripple loads as-is. Trustwright adds source links on every
// node and edge, free-form attributes, and its own node and edge kinds for trust.
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Engine names this format writes and accepts.
const (
	Engine       = "trustwright/0.1"
	RippleEngine = "ripple/0.1"
)

// Graph is the whole map. Field names and JSON keys match Ripple's export.
type Graph struct {
	Engine   string          `json:"engine"`
	Source   Origin          `json:"source"`
	Nodes    []Node          `json:"nodes"`
	Edges    []Edge          `json:"edges"`
	Levers   []Lever         `json:"levers,omitempty"`
	Effects  []Effect        `json:"effects,omitempty"`
	Snapshot json.RawMessage `json:"snapshot,omitempty"` // Ripple UI state, kept as-is
	Notes    json.RawMessage `json:"notes,omitempty"`    // Ripple reader notes, kept as-is
}

// Origin says what was read to build the graph.
type Origin struct {
	Type   string  `json:"type"` // github, zip, sample, account, inside, local
	Name   string  `json:"name"`
	URL    *string `json:"url"`
	Commit *string `json:"commit"`
}

// Node kinds. Ripple writes "file" and "package"; the rest are Trustwright's.
const (
	KindFile    = "file"
	KindPackage = "package"

	KindWorker      = "worker"      // a process or program that sends or receives messages
	KindCoordinator = "coordinator" // a process that drives workers
	KindFrame       = "frame"       // a message or frame type on the wire
	KindCheck       = "check"       // a check the code performs (CRC, signature, range, NaN...)
	KindAgent       = "agent"       // an AI agent or other actor holding permissions
	KindTool        = "tool"        // something an agent can call
	KindServer      = "server"      // a federated server or service
	KindKey         = "key"         // a key, token or other credential
	KindSpec        = "spec"        // a written rule: a spec, contract or policy section
	KindTest        = "test"        // a test case
)

// Node is one object on the map.
type Node struct {
	ID     string            `json:"id"`
	Kind   string            `json:"kind"`
	Type   string            `json:"type,omitempty"` // Ripple's sub-type: function, repo, tool, crc32, hmac...
	Group  string            `json:"group,omitempty"`
	Label  string            `json:"label,omitempty"`
	Health *Health           `json:"health,omitempty"`
	Src    []SourceRef       `json:"src,omitempty"`
	Attrs  map[string]string `json:"attrs,omitempty"`
}

// Health is Ripple's ring colour with its reason.
type Health struct {
	Level string `json:"level"` // ok, warn, quiet
	Why   string `json:"why"`
}

// Edge kinds. Ripple writes imports, uses, runs, loads, links, copies and suggested.
const (
	EdgeImports   = "imports"
	EdgeUses      = "uses"
	EdgeRuns      = "runs"
	EdgeLoads     = "loads"
	EdgeLinks     = "links"
	EdgeCopies    = "copies"
	EdgeSuggested = "suggested"

	EdgeSends      = "sends"      // worker -> frame
	EdgeAccepts    = "accepts"    // worker -> frame
	EdgeChecks     = "checks"     // check -> frame or field it guards
	EdgePerforms   = "performs"   // worker -> check it runs before accepting
	EdgeGrants     = "grants"     // agent -> tool or scope
	EdgeDelegates  = "delegates"  // agent -> agent
	EdgeSigns      = "signs"      // key -> frame
	EdgeImplements = "implements" // code -> spec
	EdgeDocuments  = "documents"  // spec -> the thing it describes
	EdgeTests      = "tests"      // test -> what it covers
)

// Edge is a directed link between two nodes.
type Edge struct {
	From  string            `json:"from"`
	To    string            `json:"to"`
	Kind  string            `json:"kind"`
	Why   string            `json:"why,omitempty"`
	Src   []SourceRef       `json:"src,omitempty"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

// Lever is something the user can switch to see its effect.
type Lever struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"` // switch
	Target string `json:"target"`
}

// Effect says what a lever does to the map.
type Effect struct {
	When      string `json:"when"`      // lever.on
	Rule      string `json:"rule"`      // reach
	From      string `json:"from"`      // lever.target
	Direction string `json:"direction"` // upstream, downstream
	Mark      string `json:"mark"`      // breaks
}

// SourceRef points at the lines that justify a node, edge or rule.
// Path is relative to the repository root, with forward slashes.
type SourceRef struct {
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`   // first line, 1-based; 0 = whole file
	End    int    `json:"end,omitempty"`    // last line, inclusive; 0 = same as Line
	Commit string `json:"commit,omitempty"` // empty = the graph's Source.Commit
	Quote  string `json:"quote,omitempty"`  // short excerpt, for specs
}

// String gives path:line or path:line-end.
func (s SourceRef) String() string {
	switch {
	case s.Line == 0:
		return s.Path
	case s.End == 0 || s.End == s.Line:
		return fmt.Sprintf("%s:%d", s.Path, s.Line)
	default:
		return fmt.Sprintf("%s:%d-%d", s.Path, s.Line, s.End)
	}
}

// Validate checks the reference on its own.
func (s SourceRef) Validate() error {
	switch {
	case s.Path == "":
		return fmt.Errorf("source ref: empty path")
	case strings.HasPrefix(s.Path, "/") || strings.Contains(s.Path, `\`):
		return fmt.Errorf("source ref %q: path must be relative with forward slashes", s.Path)
	case hasDotDot(s.Path):
		return fmt.Errorf("source ref %q: path leaves the repository", s.Path)
	case s.Line < 0 || s.End < 0:
		return fmt.Errorf("source ref %s: negative line", s)
	case s.Line == 0 && s.End != 0:
		return fmt.Errorf("source ref %s: end line without start line", s)
	case s.End != 0 && s.End < s.Line:
		return fmt.Errorf("source ref %s: end before start", s)
	}
	return nil
}

func hasDotDot(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// Decode reads a graph in either format and validates it. Unknown keys are an
// error, so a typo in a hand-written graph is caught rather than ignored.
func Decode(r io.Reader) (*Graph, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var g Graph
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("decode graph: extra data after the graph")
	}
	for _, raw := range []*json.RawMessage{&g.Snapshot, &g.Notes} {
		if len(*raw) > 0 {
			var buf bytes.Buffer
			if err := json.Compact(&buf, *raw); err != nil {
				return nil, fmt.Errorf("decode graph: %w", err)
			}
			*raw = buf.Bytes()
		}
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// Encode writes the graph as indented JSON, marking it with Trustwright's engine.
func (g *Graph) Encode(w io.Writer) error {
	out := *g
	out.Engine = Engine
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&out); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Validate checks the graph is self-consistent. It reports every problem, not just the first.
func (g *Graph) Validate() error {
	var errs []string
	add := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }

	if g.Engine != Engine && g.Engine != RippleEngine {
		add("engine %q: want %q or %q", g.Engine, Engine, RippleEngine)
	}
	ids := make(map[string]bool, len(g.Nodes))
	for i, n := range g.Nodes {
		switch {
		case n.ID == "":
			add("node %d: empty id", i)
		case ids[n.ID]:
			add("node %q: duplicate id", n.ID)
		}
		ids[n.ID] = true
		if n.Kind == "" {
			add("node %q: empty kind", n.ID)
		}
		if n.Health != nil && !oneOf(n.Health.Level, "ok", "warn", "quiet") {
			add("node %q: health level %q", n.ID, n.Health.Level)
		}
		for _, s := range n.Src {
			if err := s.Validate(); err != nil {
				add("node %q: %v", n.ID, err)
			}
		}
	}
	for i, e := range g.Edges {
		name := fmt.Sprintf("edge %d (%s -%s-> %s)", i, e.From, e.Kind, e.To)
		if !ids[e.From] {
			add("%s: unknown from node", name)
		}
		if !ids[e.To] {
			add("%s: unknown to node", name)
		}
		if e.Kind == "" {
			add("%s: empty kind", name)
		}
		for _, s := range e.Src {
			if err := s.Validate(); err != nil {
				add("%s: %v", name, err)
			}
		}
	}
	levers := make(map[string]bool, len(g.Levers))
	for _, l := range g.Levers {
		if l.ID == "" || levers[l.ID] {
			add("lever %q: empty or duplicate id", l.ID)
		}
		levers[l.ID] = true
		if !ids[l.Target] {
			add("lever %q: unknown target %q", l.ID, l.Target)
		}
	}
	for i, ef := range g.Effects {
		if !oneOf(ef.Direction, "upstream", "downstream") {
			add("effect %d: direction %q", i, ef.Direction)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid graph:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// Unsourced lists nodes of the given kinds that have no source link. The M1 gate
// is that this is empty for workers, frames and checks.
func (g *Graph) Unsourced(kinds ...string) []string {
	var out []string
	for _, n := range g.Nodes {
		if len(n.Src) == 0 && (len(kinds) == 0 || oneOf(n.Kind, kinds...)) {
			out = append(out, n.ID)
		}
	}
	return out
}

// Node returns the node with this id, or nil.
func (g *Graph) Node(id string) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}

func oneOf(s string, set ...string) bool {
	for _, x := range set {
		if s == x {
			return true
		}
	}
	return false
}
