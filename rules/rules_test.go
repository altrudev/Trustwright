package rules

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/altrudev/trustwright/model"
)

func loadRules(t *testing.T) *RuleSet {
	t.Helper()
	f, err := os.Open("../testdata/rules-example.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rs, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

func loadGraph(t *testing.T) *model.Graph {
	t.Helper()
	f, err := os.Open("../testdata/frames-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := model.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestExampleRulesLoad(t *testing.T) {
	rs := loadRules(t)
	ops := map[string]bool{}
	for _, r := range rs.Rules {
		ops[r.Check.Op] = true
	}
	for _, op := range []string{OpEachHasEdge, OpEachAttr, OpAllPathsVia, OpSameSet} {
		if !ops[op] {
			t.Errorf("example file should use op %s", op)
		}
	}
}

func TestSelector(t *testing.T) {
	g := loadGraph(t)
	cases := []struct {
		sel  Selector
		want []string
	}{
		{Selector{Kind: "worker"}, []string{"worker:c", "worker:py"}},
		{Selector{ID: "worker:*"}, []string{"worker:c", "worker:py"}},
		{Selector{Kind: "worker", Type: "c"}, []string{"worker:c"}},
		{Selector{Attrs: map[string]string{"lang": "*"}}, []string{"worker:c", "worker:py", "spec:v2-decoders"}},
		{Selector{Attrs: map[string]string{"lang": "python"}}, []string{"worker:py", "spec:v2-decoders"}},
		{Selector{Kind: "check", Attrs: map[string]string{"purpose": "authenticate"}}, nil},
		{Selector{ID: "["}, nil},
	}
	for _, c := range cases {
		if got := c.sel.Select(g); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v: got %v, want %v", c.sel, got, c.want)
		}
	}
	if n := len((Selector{}).Select(g)); n != len(g.Nodes) {
		t.Errorf("empty selector picked %d of %d nodes", n, len(g.Nodes))
	}
}

func valid() Rule {
	return Rule{
		ID: "a.b-c", Title: "T", Status: StatusActive, Severity: SeverityHigh, Family: "A",
		Check: Check{Op: OpEachHasEdge, Subject: Selector{Kind: "worker"}, Edge: "performs", Object: &Selector{Kind: "check"}},
	}
}

func TestRuleValidate(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatalf("valid rule rejected: %v", err)
	}
	cases := []struct {
		name string
		edit func(*Rule)
		want string
	}{
		{"bad id", func(r *Rule) { r.ID = "Bad ID" }, "id"},
		{"no title", func(r *Rule) { r.Title = " " }, "title"},
		{"status", func(r *Rule) { r.Status = "on" }, "status"},
		{"severity", func(r *Rule) { r.Severity = "critical" }, "severity"},
		{"family", func(r *Rule) { r.Family = "H" }, "family"},
		{"src", func(r *Rule) { r.Src = []model.SourceRef{{Path: "../x"}} }, "leaves"},
		{"empty subject", func(r *Rule) { r.Check.Subject = Selector{} }, "subject"},
		{"bad pattern", func(r *Rule) { r.Check.Subject = Selector{ID: "["} }, "pattern"},
		{"direction", func(r *Rule) { r.Check.Direction = "up" }, "direction"},
		{"unknown op", func(r *Rule) { r.Check.Op = "exists" }, "unknown op"},
		{"edge op without edge", func(r *Rule) { r.Check.Edge = "" }, "needs edge"},
		{"edge op without object", func(r *Rule) { r.Check.Object = nil }, "needs object"},
		{"edge op with key", func(r *Rule) { r.Check.Key = "lang" }, "does not use key"},
		{"attr op without want", func(r *Rule) { r.Check = Check{Op: OpEachAttr, Subject: Selector{Kind: "x"}} }, "needs want"},
		{"attr op with edge", func(r *Rule) {
			r.Check = Check{Op: OpEachAttr, Subject: Selector{Kind: "x"}, Want: map[string]string{"a": "b"}, Edge: "e"}
		}, "does not use edge"},
		{"path op without via", func(r *Rule) {
			r.Check = Check{Op: OpAllPathsVia, Subject: Selector{Kind: "x"}, Object: &Selector{Kind: "y"}}
		}, "needs via"},
		{"path op with empty via", func(r *Rule) {
			r.Check = Check{Op: OpAllPathsVia, Subject: Selector{Kind: "x"}, Object: &Selector{Kind: "y"}, Via: &Selector{}}
		}, "needs via"},
		{"same-set without key", func(r *Rule) {
			r.Check = Check{Op: OpSameSet, Subject: Selector{Kind: "x"}, Object: &Selector{Kind: "y"}}
		}, "needs key"},
	}
	for _, c := range cases {
		r := valid()
		c.edit(&r)
		err := r.Validate()
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want error mentioning %q, got %v", c.name, c.want, err)
		}
	}
}

func TestRuleSetValidate(t *testing.T) {
	rs := &RuleSet{Format: "rules/0", Rules: []Rule{valid(), valid()}}
	err := rs.Validate()
	if err == nil || !strings.Contains(err.Error(), "format") || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("want format and duplicate errors, got %v", err)
	}
	_, err = Decode(strings.NewReader(`{"format":"trustwright.rules/1","rules":[],"extra":1}`))
	if err == nil {
		t.Fatal("want unknown field error")
	}
}
