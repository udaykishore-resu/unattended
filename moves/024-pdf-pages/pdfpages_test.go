package pdfpages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestRowsTaggedWithPage(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{
	  "header":"Lanes 401-450","continues_from_prev":false,
	  "rows":[{"origin":"30301","dest":"75201","rate_cents":185000},
	          {"origin":"30301","dest":"77001","rate_cents":172500}]}`}}}
	p, err := onePage(context.Background(), f, 9, llm.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Rows) != 2 || p.Rows[0].Page != 9 || p.Rows[1].Page != 9 {
		t.Fatalf("rows = %+v", p.Rows)
	}
}

// A page that stopped partway (the "388 of 612"
// failure) is an error, never data.
func TestCutOffPageIsAnError(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{
		Text: `{"rows":[{"origin":"30301"`, StopReason: llm.StopMaxTokens}}}
	_, err := onePage(context.Background(), f, 9, llm.Request{})
	if err == nil || !strings.Contains(err.Error(), "page 9: stopped: max_tokens") {
		t.Fatalf("got %v", err)
	}
}

func TestBadJSONNamesThePage(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: "Sure!"}}}
	_, err := onePage(context.Background(), f, 11, llm.Request{})
	if err == nil || !strings.HasPrefix(err.Error(), "page 11:") {
		t.Fatalf("got %v", err)
	}
}

// Verification: the full path validates after decode.
// Schema-level (unknown field, wrong type, trailing
// data) and business-level (bad ZIP, rate out of
// range) failures both wrap ErrInvalid.
func TestParsePageValidates(t *testing.T) {
	for name, in := range map[string]string{
		"unknown field": `{"rows":[],"lane_count":12}`,
		"wrong type":    `{"rows":[{"origin":"30301","dest":"75201","rate_cents":"1,850"}]}`,
		"trailing":      `{"rows":[]} {"rows":[]}`,
		"bad zip":       `{"rows":[{"origin":"Atlanta","dest":"75201","rate_cents":185000}]}`,
		"zero rate":     `{"rows":[{"origin":"30301","dest":"75201","rate_cents":0}]}`,
		"huge rate":     `{"rows":[{"origin":"30301","dest":"75201","rate_cents":99999999999}]}`,
		"prose":         `Sure! Here is page 9.`,
	} {
		if _, err := parsePage(in); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

func row(o, d string, c int64) string {
	return fmt.Sprintf(`{"origin":%q,"dest":%q,"rate_cents":%d}`, o, d, c)
}

// One repair for an invalid page, then review; no
// third call. Stop reasons and transport errors are
// not repaired.
func TestOneRepairPerPage(t *testing.T) {
	bad := `{"rows":[` + row("3030", "75201", 185000) + `]}`
	good := `{"rows":[` + row("30301", "75201", 185000) + `]}`
	f := &llm.FakeModel{Responses: []llm.Response{{Text: bad}, {Text: good}}}
	p, err := ExtractPage(context.Background(), f, 4, llm.Request{})
	if err != nil || len(p.Rows) != 1 || p.Rows[0].Page != 4 || len(f.Calls) != 2 {
		t.Fatalf("repaired: p=%+v err=%v calls=%d", p, err, len(f.Calls))
	}
	last := f.Calls[1].Messages[len(f.Calls[1].Messages)-1]
	if !strings.Contains(last.Content, "not five digits") {
		t.Fatalf("repair note = %q", last.Content)
	}

	f = &llm.FakeModel{Responses: []llm.Response{{Text: bad}, {Text: bad}, {Text: good}}}
	if _, err := ExtractPage(context.Background(), f, 4, llm.Request{}); !errors.Is(err, ErrInvalid) || len(f.Calls) != 2 {
		t.Fatalf("twice bad: err=%v calls=%d", err, len(f.Calls))
	}

	f = &llm.FakeModel{Responses: []llm.Response{{Text: bad, StopReason: llm.StopMaxTokens}}}
	if _, err := ExtractPage(context.Background(), f, 4, llm.Request{}); errors.Is(err, ErrInvalid) || len(f.Calls) != 1 {
		t.Fatalf("cut off: err=%v calls=%d", err, len(f.Calls))
	}
}

func page(header string, cont bool, rows ...Row) PageOut {
	return PageOut{Header: header, Rows: rows, ContinuesFromPrev: cont}
}

func r(o, d string, c int64, pg int) Row {
	return Row{Origin: o, Dest: d, RateCents: c, Page: pg}
}

func TestStitch(t *testing.T) {
	h := "Origin Dest Rate"
	A, B := r("30301", "75201", 1, 1), r("30301", "77001", 2, 1)
	C1, C2 := r("30301", "78201", 3, 1), r("30301", "78201", 3, 2)
	D := r("30301", "79901", 4, 2)

	// Row cut at the page break, returned whole by
	// page 2 with the flag; page 1 left it out.
	rows, probs := Stitch(map[int]PageOut{
		1: page(h, false, A, B),
		2: page(h, true, C2, D),
	}, 2, 4)
	if len(probs) != 0 || len(rows) != 4 {
		t.Fatalf("split row: %v %v", rows, probs)
	}
	// Both pages returned it: keep one copy.
	rows, probs = Stitch(map[int]PageOut{
		1: page(h, false, A, B, C1),
		2: page(h, true, C2, D),
	}, 2, 4)
	if len(probs) != 0 || len(rows) != 4 || rows[2].Page != 1 {
		t.Fatalf("double: %v %v", rows, probs)
	}
	// Repeated header (same text, other spacing) is fine;
	// a page without a printed header is fine.
	_, probs = Stitch(map[int]PageOut{
		1: page(h, false, A), 2: page("Origin  Dest\nRate", false, B),
		3: page("", false, D),
	}, 3, 0)
	if len(probs) != 0 {
		t.Fatalf("repeated header: %v", probs)
	}

	cases := map[string]struct {
		pages   map[int]PageOut
		count   int
		printed int
		want    string
	}{
		"missing page": {map[int]PageOut{1: page(h, false, A), 3: page(h, false, D)}, 3, 0, "page 2: missing"},
		"empty page":   {map[int]PageOut{1: page(h, false, A), 2: page(h, false)}, 2, 0, "page 2: no rows"},
		"new header":   {map[int]PageOut{1: page(h, false, A), 2: page("Lane Miles", false, B)}, 2, 0, "header changed"},
		"orphan cont":  {map[int]PageOut{2: page(h, true, C2)}, 2, 0, "page 2: continues a missing page"},
		"count short":  {map[int]PageOut{1: page(h, false, A, B)}, 1, 3, "rows 2 != printed count 3"},
	}
	for name, c := range cases {
		_, probs := Stitch(c.pages, c.count, c.printed)
		if !strings.Contains(strings.Join(probs, "|"), c.want) {
			t.Errorf("%s: probs = %v, want %q", name, probs, c.want)
		}
	}
}
