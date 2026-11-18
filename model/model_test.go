package model

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func load(t *testing.T, name string) *Graph {
	t.Helper()
	f, err := os.Open("../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestRippleExportLoads(t *testing.T) {
	g := load(t, "ripple-export.json")
	if g.Engine != RippleEngine || len(g.Nodes) != 3 || len(g.Edges) != 3 || len(g.Levers) != 1 || len(g.Effects) != 1 {
		t.Fatalf("unexpected shape: %+v", g)
	}
	if h := g.Node("src/app.js").Health; h == nil || h.Level != "ok" {
		t.Errorf("health not read: %+v", h)
	}
	if g.Edges[2].Kind != EdgeSuggested || g.Edges[2].Why == "" {
		t.Errorf("ghost edge not read: %+v", g.Edges[2])
	}
	if len(g.Snapshot) == 0 || len(g.Notes) == 0 {
		t.Error("snapshot and notes should be kept")
	}
}

func TestRoundTrip(t *testing.T) {
	for _, name := range []string{"ripple-export.json", "frames-graph.json"} {
		g := load(t, name)
		var buf bytes.Buffer
		if err := g.Encode(&buf); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), `"engine": "trustwright/0.1"`) {
			t.Errorf("%s: encoded graph should say engine %s", name, Engine)
		}
		back, err := Decode(&buf)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		back.Engine = g.Engine
		if !reflect.DeepEqual(g, back) {
			t.Errorf("%s: round trip changed the graph\nbefore %+v\nafter  %+v", name, g, back)
		}
	}
}

func TestUnsourced(t *testing.T) {
	g := load(t, "frames-graph.json")
	if got := g.Unsourced(KindWorker, KindFrame, KindCheck); len(got) != 0 {
		t.Errorf("all workers, frames and checks have sources, got %v", got)
	}
	g.Nodes = append(g.Nodes, Node{ID: "check:nan", Kind: KindCheck})
	if got := g.Unsourced(KindWorker, KindFrame, KindCheck); !reflect.DeepEqual(got, []string{"check:nan"}) {
		t.Errorf("got %v", got)
	}
	if got := g.Unsourced(); len(got) != 1 {
		t.Errorf("no kinds should mean all kinds, got %v", got)
	}
}

func TestValidateReportsEveryProblem(t *testing.T) {
	g := &Graph{
		Engine: "other/1",
		Nodes: []Node{
			{ID: "a", Kind: "file"},
			{ID: "a", Kind: ""},
			{ID: "b", Kind: "check", Health: &Health{Level: "red"}, Src: []SourceRef{{Path: "/etc/passwd"}}},
		},
		Edges:   []Edge{{From: "a", To: "zz", Kind: "uses"}},
		Levers:  []Lever{{ID: "l", Target: "nope"}},
		Effects: []Effect{{Direction: "sideways"}},
	}
	err := g.Validate()
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{"engine", "duplicate id", "empty kind", "health level", "relative", "unknown to node", "unknown target", "direction"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q:\n%v", want, err)
		}
	}
}

func TestDecodeRejectsUnknownKeys(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"engine":"trustwright/0.1","source":{},"nodes":[{"id":"a","kind":"file","colour":"red"}],"edges":[]}`))
	if err == nil || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("want unknown field error, got %v", err)
	}
	_, err = Decode(strings.NewReader(`{"engine":"trustwright/0.1","source":{},"nodes":[],"edges":[]} {}`))
	if err == nil {
		t.Fatal("want error for trailing data")
	}
}

func TestSourceRef(t *testing.T) {
	cases := []struct {
		ref  SourceRef
		str  string
		okay bool
	}{
		{SourceRef{Path: "a.go"}, "a.go", true},
		{SourceRef{Path: "a.go", Line: 3}, "a.go:3", true},
		{SourceRef{Path: "a.go", Line: 3, End: 3}, "a.go:3", true},
		{SourceRef{Path: "a.go", Line: 3, End: 9}, "a.go:3-9", true},
		{SourceRef{Path: "a.go", Line: 9, End: 3}, "a.go:9-3", false},
		{SourceRef{Path: "a.go", End: 3}, "a.go", false},
		{SourceRef{Path: "../x.go"}, "../x.go", false},
		{SourceRef{Path: `dir\x.go`}, `dir\x.go`, false},
		{SourceRef{Path: "a..b.go"}, "a..b.go", true},
		{SourceRef{}, "", false},
	}
	for _, c := range cases {
		if got := c.ref.String(); got != c.str {
			t.Errorf("%+v: String() = %q, want %q", c.ref, got, c.str)
		}
		if err := c.ref.Validate(); (err == nil) != c.okay {
			t.Errorf("%+v: Validate() = %v, want ok=%v", c.ref, err, c.okay)
		}
	}
}
