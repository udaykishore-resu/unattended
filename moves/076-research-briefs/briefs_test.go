package briefs

import (
	"slices"
	"testing"
)

func TestBadCites(t *testing.T) {
	brief := "Their CFO joined in 2022 [S2]. They opened a Dallas DC [S3]. " +
		"They run 40 lanes with us [S1]. Their CFO is new [S7]. Zero source [S0]."
	got := BadCites(brief, 3)
	if !slices.Equal(got, []string{"[S7]", "[S0]"}) {
		t.Fatalf("got %v", got)
	}
}
