package cite

import "testing"

// sent holds chunk bodies only. A quote that exists
// only in the contextual header (Move 43) or in the
// model-written situating sentences is not evidence.
func TestHeaderTextIsNotEvidence(t *testing.T) {
	body := "Two free hours, then $75 per hour, billed in 15-minute steps."
	header := "[Brightline Foods Agreement > Detention > Unloading | updated 2026-02-01]"
	situated := "Unloading detention terms for Brightline Foods."
	_ = header + situated // what the embedder saw
	sent := map[string]string{"c-17": body}
	for _, q := range []string{
		"Brightline Foods Agreement > Detention",
		"Unloading detention terms for Brightline Foods",
	} {
		a := Answer{Citations: []Citation{{"c-17", q}}}
		if Check(a, sent) == nil {
			t.Errorf("header quote %q accepted as evidence", q)
		}
	}
	ok := Answer{Citations: []Citation{{"c-17", "then $75 per hour, billed"}}}
	if err := Check(ok, sent); err != nil {
		t.Fatal(err)
	}
}
