// Package pdfpages extracts a table one page per
// call and tags each row with its page.
//
// Move 24: extract tables from PDFs page by page.
package pdfpages

import (
	"context"
	"encoding/json"
	"fmt"

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
	var p PageOut
	err = json.Unmarshal([]byte(resp.Text), &p)
	if err != nil {
		return PageOut{}, fmt.Errorf("page %d: %w", n, err)
	}
	for i := range p.Rows {
		p.Rows[i].Page = n
	}
	return p, nil
}
