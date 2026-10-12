package templates

import (
	"strings"
	"testing"
	"text/template"
)

func TestRender(t *testing.T) {
	out, err := RenderClassify(ClassifyVars{
		From: "carrier@example.com", Subject: "Late truck",
		Body: "Truck for HF-1 is late", OpenLoads: []string{"HF-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<email>", "Late truck", "HF-1 "} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestEmptyBodyRefused(t *testing.T) {
	if _, err := RenderClassify(ClassifyVars{Body: "  \n"}); err == nil {
		t.Fatal("an empty body (a PDF with no text) must not reach the model")
	}
}

// A misspelled field is an execute-time error with a
// struct. With a map it would print <no value>.
func TestMisspelledFieldFails(t *testing.T) {
	bad := template.Must(template.New("x").
		Option("missingkey=error").Parse("{{.Sbject}}"))
	err := bad.Execute(new(strings.Builder), ClassifyVars{Subject: "s"})
	if err == nil || !strings.Contains(err.Error(), "Sbject") {
		t.Fatalf("want a field error, got %v", err)
	}
	// The map trap the book warns about, without missingkey=error.
	var b strings.Builder
	loose := template.Must(template.New("y").Parse("{{.Sbject}}"))
	loose.Execute(&b, map[string]string{"Subject": "s"})
	if b.String() != "<no value>" {
		t.Fatalf("map render = %q", b.String())
	}
}

func TestEveryFieldReachesTheModel(t *testing.T) {
	v := ClassifyVars{From: "M-FROM", Subject: "M-SUBJ",
		Body: "M-BODY", OpenLoads: []string{"M-LOAD"}}
	out, err := RenderClassify(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"M-FROM", "M-SUBJ", "M-BODY", "M-LOAD"} {
		if !strings.Contains(out, m) {
			t.Errorf("field marker %s missing from prompt", m)
		}
	}
}

func TestClipOnRuneBoundary(t *testing.T) {
	got := clip("héllo", 2)
	if got != "hé[truncated]" {
		t.Fatalf("clip = %q", got)
	}
}
