package planreview

import (
	"encoding/json"
	"strings"
	"testing"
)

const plan = `{
  "prior_state": {"values": {"password": "hunter2-prior"}},
  "variables": {"db_password": {"value": "hunter2-var"}},
  "resource_changes": [
    {"address": "aws_db_instance.dispatch",
     "change": {"actions": ["update"],
       "before": {"password": "hunter2-old", "instance_class": "db.m5.large",
                  "tags": [{"k": "env"}, {"k": "secret-tag"}]},
       "after":  {"password": "hunter2-new", "instance_class": "db.m5.xlarge",
                  "tags": [{"k": "env"}, {"k": "secret-tag"}]},
       "before_sensitive": {"password": true, "tags": [false, true]},
       "after_sensitive":  {"password": true, "tags": [false, true]}}},
    {"address": "aws_s3_bucket.logs",
     "change": {"actions": ["no-op"], "before": {}, "after": {}}},
    {"address": "kubernetes_deployment.dispatch",
     "change": {"actions": ["delete", "create"],
       "before": {"replicas": 3}, "after": {"replicas": 1}}}
  ]
}`

func TestTrimAndMask(t *testing.T) {
	cs, err := ResourceChanges([]byte(plan))
	if err != nil {
		t.Fatal(err)
	}
	got := Trim(cs)
	if len(got) != 2 {
		t.Fatalf("no-op not skipped: %d changes", len(got))
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(got)
	out := b.String()
	if strings.Contains(out, "hunter2") || strings.Contains(out, "secret-tag") {
		t.Fatalf("secret leaked:\n%s", out)
	}
	for _, want := range []string{`"<sensitive>"`, "db.m5.xlarge", `"k":"env"`,
		`["delete","create"]`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
	if strings.Contains(out, "before_sensitive") && strings.Contains(out, "true") {
		t.Errorf("sensitive trees should be dropped:\n%s", out)
	}
}
