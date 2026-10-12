// Package pdfpages extracts a table one page per
// call, validates each page, and stitches the pages
// back together in code.
//
// Move 24: extract tables from PDFs page by page.
package pdfpages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

// PageOut holds the header as printed, the rows,
// and whether the first row finishes a row split
// at the bottom of the last page.
type PageOut struct {
	Header            string `json:"header"`
	Rows              []Row  `json:"rows"`
	ContinuesFromPrev bool   `json:"continues_from_prev"`
}

// Row is one lane on a rate sheet. Page is set in
// code, never by the model.
type Row struct {
	Origin    string `json:"origin"`
	Dest      string `json:"dest"`
	RateCents int64  `json:"rate_cents"`
	Page      int    `json:"-"`
}

// ErrInvalid marks a page that decoded badly or broke
// a row rule. It gets one repair (Move 21), then
// goes to review; other errors do not.
var ErrInvalid = errors.New("invalid page")

func onePage(ctx context.Context, m llm.Model,
	n int, req llm.Request) (PageOut, error) {
	resp, err := m.Complete(ctx, req)
	if err != nil {
		return PageOut{}, err
	}
	if resp.StopReason != llm.StopEnd {
		return PageOut{}, fmt.Errorf(
			"page %d: stopped: %s", n, resp.StopReason)
	}
	p, err := parsePage(resp.Text)
	if err != nil {
		return PageOut{}, fmt.Errorf("page %d: %w", n, err)
	}
	for i := range p.Rows {
		p.Rows[i].Page = n
	}
	return p, nil
}

var zip5 = regexp.MustCompile(`^[0-9]{5}$`)

// maxRateCents is Halcyon's sanity bound for one lane
// ($100,000), a business rule rather than a law.
const maxRateCents = 100_000_00

// parsePage is the schema check (strict decode, as in
// Move 20) plus the business rules for each row.
// Every failure wraps ErrInvalid.
func parsePage(text string) (PageOut, error) {
	var p PageOut
	dec := json.NewDecoder(strings.NewReader(text))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return PageOut{}, fmt.Errorf("%w: decode: %v",
			ErrInvalid, err)
	}
	if dec.More() {
		return PageOut{}, fmt.Errorf(
			"%w: trailing data", ErrInvalid)
	}
	var probs []string
	for i, r := range p.Rows {
		if !zip5.MatchString(r.Origin) ||
			!zip5.MatchString(r.Dest) {
			probs = append(probs, fmt.Sprintf(
				"row %d: origin/dest not five digits", i+1))
		}
		if r.RateCents < 1 || r.RateCents > maxRateCents {
			probs = append(probs, fmt.Sprintf(
				"row %d: rate_cents %d out of range",
				i+1, r.RateCents))
		}
	}
	if len(probs) > 0 {
		return PageOut{}, fmt.Errorf("%w: %s",
			ErrInvalid, strings.Join(probs, "; "))
	}
	return p, nil
}

// ExtractPage runs onePage and, for an ErrInvalid
// page only, sends one repair that names the errors.
// A second failure is returned for review; there is
// no third call. Transport errors and stop reasons
// are returned as they are, for the retry wrapper or
// the caller.
func ExtractPage(ctx context.Context, m llm.Model,
	n int, req llm.Request) (PageOut, error) {
	resp, err := m.Complete(ctx, req)
	if err != nil {
		return PageOut{}, err
	}
	p, err := checkResp(n, resp)
	if !errors.Is(err, ErrInvalid) {
		return p, err
	}
	req.Messages = append(req.Messages,
		llm.Message{Role: "assistant", Content: resp.Text},
		llm.Message{Role: "user", Content: "This page " +
			"failed validation: " + err.Error() +
			". Return the whole page again, fixed. If a " +
			"value is not on the page, leave the row out."})
	return onePage(ctx, m, n, req)
}

func checkResp(n int, resp llm.Response) (PageOut, error) {
	if resp.StopReason != llm.StopEnd {
		return PageOut{}, fmt.Errorf(
			"page %d: stopped: %s", n, resp.StopReason)
	}
	p, err := parsePage(resp.Text)
	if err != nil {
		return PageOut{}, fmt.Errorf("page %d: %w", n, err)
	}
	for i := range p.Rows {
		p.Rows[i].Page = n
	}
	return p, nil
}

// Stitch joins pages 1..count in order. A row cut
// off at the bottom of a page is left out there; the
// next page, which sees the last three lines of the
// previous one for context, returns it whole and sets
// ContinuesFromPrev. Stitch drops that row if the
// previous page returned it too.
//
// It returns one problem per gap it can see: a
// missing page, an empty page, a header that changed
// between pages, a continuation with no page before
// it, and a row total that differs from the printed
// count (printed is 0 when the document prints none).
// Any problem sends the document to review.
func Stitch(pages map[int]PageOut, count, printed int) (
	[]Row, []string) {
	var rows []Row
	var probs []string
	header := ""
	for n := 1; n <= count; n++ {
		p, ok := pages[n]
		if !ok {
			probs = append(probs,
				fmt.Sprintf("page %d: missing", n))
			continue
		}
		if len(p.Rows) == 0 {
			probs = append(probs,
				fmt.Sprintf("page %d: no rows", n))
			continue
		}
		h := strings.Join(strings.Fields(p.Header), " ")
		if header == "" {
			header = h
		} else if h != "" && h != header {
			probs = append(probs, fmt.Sprintf(
				"page %d: header changed: %q", n, h))
		}
		body := p.Rows
		if p.ContinuesFromPrev {
			_, prevOK := pages[n-1]
			switch {
			case n == 1 || !prevOK:
				probs = append(probs, fmt.Sprintf(
					"page %d: continues a missing page", n))
			case len(rows) > 0 &&
				sameLane(rows[len(rows)-1], body[0]):
				body = body[1:] // both pages returned it
			}
		}
		rows = append(rows, body...)
	}
	if printed > 0 && len(rows) != printed {
		probs = append(probs, fmt.Sprintf(
			"rows %d != printed count %d",
			len(rows), printed))
	}
	return rows, probs
}

func sameLane(a, b Row) bool {
	return a.Origin == b.Origin && a.Dest == b.Dest &&
		a.RateCents == b.RateCents
}
