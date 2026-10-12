package headers

import (
	"testing"
	"time"
)

func TestCheckMeta(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	good := Chunk{ID: "c1", DocTitle: "Brightline Foods Agreement",
		Path: []string{"Detention"}, Updated: now.AddDate(0, -1, 0)}
	if err := CheckMeta(good, now); err != nil {
		t.Fatal(err)
	}
	bad := map[string]Chunk{
		"no title":    {ID: "c2", Path: good.Path, Updated: good.Updated},
		"no trail":    {ID: "c3", DocTitle: good.DocTitle, Updated: good.Updated},
		"zero date":   {ID: "c4", DocTitle: good.DocTitle, Path: good.Path},
		"future date": {ID: "c5", DocTitle: good.DocTitle, Path: good.Path, Updated: now.AddDate(1, 0, 0)},
	}
	for name, c := range bad {
		if CheckMeta(c, now) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
