package emailrecord

import (
	"errors"
	"testing"
)

const box = "ops@"

func TestKeyScope(t *testing.T) {
	id := "<CAF+1x@mail.example.com>"
	k0, err := Key(box, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := Key(box, " "+id+" ", 0) // redelivered, header folded
	k1, _ := Key(box, id, 1)            // second tender, same email
	other, _ := Key("claims@", id, 0)   // same email, other mailbox
	if k0 != again {
		t.Errorf("redelivery changed the key: %q vs %q", k0, again)
	}
	if k0 == k1 || k0 == other {
		t.Errorf("keys collide: %q %q %q", k0, k1, other)
	}
}

func TestBadMessageIDGoesToReview(t *testing.T) {
	for _, id := range []string{
		"", "   ", "abc@def", "<>", "<@x>", "<x@>", "<a@b@c>",
		"<a b@c>", "<a@b", "a@b>", "<<a@b>>",
	} {
		if _, err := Key(box, id, 0); !errors.Is(err, ErrNoMessageID) {
			t.Errorf("%q: err = %v", id, err)
		}
	}
	if _, err := Key(box, "<a@b>", -1); err == nil {
		t.Error("negative position accepted")
	}
}

// Duplicate delivery: the second run finds the key and
// does not write. A change_to_existing email is keyed
// the same way: the key marks the email applied.
func TestDuplicateDeliveryWritesOnce(t *testing.T) {
	var in Intake
	writes := 0
	upd := func() (State, error) { writes++; return Updated, nil }
	k, _ := Key(box, "<r2@acme.example>", 0)
	for range 3 {
		st, _, err := in.Apply(k, upd)
		if err != nil || st != Updated {
			t.Fatalf("st=%v err=%v", st, err)
		}
	}
	if writes != 1 {
		t.Fatalf("writes = %d, want 1", writes)
	}
}

// Three tenders in one email: three keys, three writes,
// and a redelivery of the whole email writes none.
func TestMultipleTendersInOneEmail(t *testing.T) {
	var in Intake
	writes := 0
	create := func() (State, error) { writes++; return Created, nil }
	for range 2 { // delivered twice
		for i := range 3 {
			k, err := Key(box, "<t3@acme.example>", i)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := in.Apply(k, create); err != nil {
				t.Fatal(err)
			}
		}
	}
	if writes != 3 {
		t.Fatalf("writes = %d, want 3", writes)
	}
}

// A failed write is not recorded, so a retry can try
// again instead of replaying a state that never existed.
func TestFailedWriteNotRecorded(t *testing.T) {
	var in Intake
	k, _ := Key(box, "<f@acme.example>", 0)
	if _, _, err := in.Apply(k, func() (State, error) {
		return "", errors.New("db down")
	}); err == nil {
		t.Fatal("want error")
	}
	st, replay, err := in.Apply(k, func() (State, error) { return Created, nil })
	if err != nil || replay || st != Created {
		t.Fatalf("st=%v replay=%v err=%v", st, replay, err)
	}
}
