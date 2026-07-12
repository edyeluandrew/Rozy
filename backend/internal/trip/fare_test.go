package trip

import (
	"testing"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to Status
		ok       bool
	}{
		{StatusRequested, StatusSearching, true},
		{StatusSearching, StatusDriverAssigned, true},
		{StatusDriverArriving, StatusInProgress, true},
		{StatusInProgress, StatusCompleted, true},
		{StatusCompleted, StatusSearching, false},
		{StatusSearching, StatusInProgress, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.ok {
			t.Errorf("%s -> %s: got %v want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestCalculateFare(t *testing.T) {
	rule := FareRule{
		BaseFare:  1500,
		PerKmRate: 500,
		MinFare:   2000,
		RoundTo:   500,
		MinBillableKm: 0.5,
	}

	// 3 km boda: 1500 + 1500 = 3000
	if fare := CalculateFare(rule, 3); fare != 3000 {
		t.Fatalf("3km fare = %d, want 3000", fare)
	}

	// 0.2 km -> min billable 0.5 -> 1500+250=1750 -> min 2000
	if fare := CalculateFare(rule, 0.2); fare != 2000 {
		t.Fatalf("short trip fare = %d, want 2000", fare)
	}

	// 4 km: 1500+2000=3500 (already on 500 boundary)
	if fare := CalculateFare(rule, 4); fare != 3500 {
		t.Fatalf("4km fare = %d, want 3500", fare)
	}
}
