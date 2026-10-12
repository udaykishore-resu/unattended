package codeqa

import "testing"

var got = []Chunk{{Path: "dispatch/score.go",
	Commit: "9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d", Start: 30, End: 60}}

func TestCheck(t *testing.T) {
	cases := map[string]bool{
		"Carriers are scored in ScoreCarrier [dispatch/score.go:41@9c1d2e3].": true,
		"Carriers are scored somewhere in dispatch.":                          false, // no citation
		"See [dispatch/score.go:99@9c1d2e3].":                                 false, // line outside chunk
		"See [dispatch/score.go:41@abcdef1].":                                 false, // other commit
		"See [ratequote/region.go:12@9c1d2e3].":                               false, // not retrieved
	}
	for ans, ok := range cases {
		if err := Check(ans, got); (err == nil) != ok {
			t.Errorf("%q: err = %v", ans, err)
		}
	}
}
