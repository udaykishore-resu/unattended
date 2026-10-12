//go:build integration

package hybrid

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Runs VectorSQL and KeywordSQL verbatim in Postgres
// through psql's \bind (psql 16+). This sandbox has
// no pgvector, so the schema stands in a domain
// named vector over float8[] and a cosine <=>
// operator; the query text is unchanged.
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test -tags integration ./moves/044-hybrid-search/
func psql(t *testing.T, url, script string) string {
	t.Helper()
	cmd := exec.Command("psql", url, "-X", "-q", "-A", "-t",
		"-v", "ON_ERROR_STOP=1")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql: %v\n%s", err, out)
	}
	return string(out)
}

const schema = `
CREATE DOMAIN vector AS float8[];
CREATE FUNCTION cosdist(a float8[], b float8[])
RETURNS float8 LANGUAGE sql IMMUTABLE AS $$
  SELECT 1 - (a[1]*b[1] + a[2]*b[2]) /
    (sqrt(a[1]^2 + a[2]^2) * sqrt(b[1]^2 + b[2]^2))
$$;
CREATE OPERATOR <=> (LEFTARG = float8[],
  RIGHTARG = float8[], FUNCTION = cosdist);
CREATE TABLE chunks (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  body text NOT NULL,
  expires_at timestamptz,
  embedding vector NOT NULL,
  tsv tsvector GENERATED ALWAYS AS
    (to_tsvector('english', body)) STORED);
INSERT INTO chunks VALUES
 ('bl-detention', 'brightline',
  'Brightline detention: two free hours, then $75 per hour.',
  NULL, '{1,0}'),
 ('bl-fuel-old', 'brightline',
  'Brightline fuel surcharge detention table, 31 cents.',
  now() - interval '1 day', '{1,0.01}'),
 ('bl-fuel-new', 'brightline',
  'Brightline fuel surcharge table, 34 cents.',
  now() + interval '30 days', '{0.9,0.1}'),
 ('acme-detention', 'acme',
  '[Brightline Foods > Detention] Acme detention terms.',
  NULL, '{1,0}');
`

func TestQueriesFilterTenantAndExpiryInSQL(t *testing.T) {
	url := os.Getenv("UNATTENDED_TEST_PSQL_URL")
	if url == "" {
		t.Skip("UNATTENDED_TEST_PSQL_URL not set")
	}
	s := fmt.Sprintf("t%d", time.Now().UnixNano())
	psql(t, url, "CREATE SCHEMA "+s+";")
	t.Cleanup(func() {
		psql(t, url, "DROP SCHEMA "+s+" CASCADE;")
	})
	psql(t, url, "SET search_path = "+s+";"+schema)

	run := func(q, tenant, arg string) []string {
		out := psql(t, url, "SET search_path = "+s+";\n"+
			q+" \\bind '"+tenant+"' '"+arg+"' \\g\n")
		return strings.Fields(out)
	}
	// Vector side: acme's chunk is the nearest vector
	// and even names Brightline in a (misleading)
	// header, but the tenant filter keeps it out. The
	// expired fuel table never comes back.
	got := run(VectorSQL, "brightline", "{1,0}")
	want := []string{"bl-detention", "bl-fuel-new"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("vector: got %v, want %v", got, want)
	}
	// Keyword side: "detention" matches all three
	// non-fuel-new rows; only the live brightline
	// one may come back.
	got = run(KeywordSQL, "brightline", "detention")
	if !reflect.DeepEqual(got, []string{"bl-detention"}) {
		t.Errorf("keyword: got %v", got)
	}
	// And the other tenant sees only its own row.
	got = run(KeywordSQL, "acme", "brightline detention")
	if !reflect.DeepEqual(got, []string{"acme-detention"}) {
		t.Errorf("acme keyword: got %v", got)
	}
}
