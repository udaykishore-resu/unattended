package callfollowup

import (
	"errors"
	"slices"
	"testing"
)

func TestNoConsentNeverTranscribed(t *testing.T) {
	var queued []string
	enqueue = func(id string) error { queued = append(queued, id); return nil }
	defer func() { enqueue = func(string) error { return nil } }()

	if err := Start(Call{ID: "call-1"}); !errors.Is(err, ErrNoConsent) {
		t.Fatalf("got %v", err)
	}
	if err := Start(Call{ID: "call-2", ConsentRecorded: true}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(queued, []string{"call-2"}) {
		t.Fatalf("queued = %v", queued)
	}
}

// "We can do 14 loads a week starting the 3rd."
func TestChecklist(t *testing.T) {
	turns := []Turn{{Start: 400, End: 418, Side: "them",
		Text: "We can do 14 loads a week starting the 3rd."}}
	s := Summary{
		Commitments: []Commitment{
			{Who: "them", What: "14 loads a week", When: "the 3rd", Start: 412},
			{Who: "us", What: "cover detention", Start: 900}, // lands in silence
		},
		Numbers: []string{"14 loads", "the 3rd"},
	}
	want := []string{"no turn at timestamp: cover detention",
		"confirm: 14 loads", "confirm: the 3rd"}
	if got := Checks(s, turns); !slices.Equal(got, want) {
		t.Fatalf("got %q", got)
	}
}
