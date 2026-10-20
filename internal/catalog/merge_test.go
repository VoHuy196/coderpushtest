package catalog

import (
	"testing"
	"time"
)

func TestDecide(t *testing.T) {
	t1 := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	t0 := t1.Add(-time.Hour)

	active := &Product{SKU: "A", Status: StatusActive, SupplierUpdatedAt: t1}
	withdrawn := &Product{SKU: "A", Status: StatusWithdrawn, SupplierUpdatedAt: t1}

	tests := []struct {
		name      string
		existing  *Product
		at        time.Time
		withdrawn bool
		want      Action
	}{
		{"new product", nil, t1, false, ActionCreate},
		{"withdrawal for unknown sku keeps a tombstone", nil, t1, true, ActionWithdraw},
		{"same timestamp is a repeat", active, t1, false, ActionSkip},
		{"older data is ignored", active, t0, false, ActionSkip},
		{"newer data updates", active, t2, false, ActionUpdate},
		{"newer withdrawal withdraws", active, t2, true, ActionWithdraw},
		{"older withdrawal is ignored", active, t0, true, ActionSkip},
		{"older data cannot restore a withdrawn product", withdrawn, t0, false, ActionSkip},
		{"repeated data cannot restore a withdrawn product", withdrawn, t1, false, ActionSkip},
		{"newer data puts it back on sale", withdrawn, t2, false, ActionUpdate},
		{"newer withdrawal of withdrawn product refreshes timestamp", withdrawn, t2, true, ActionUpdate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := Decide(tc.existing, tc.at, tc.withdrawn)
			if got != tc.want {
				t.Errorf("Decide() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestDiff(t *testing.T) {
	old := Product{Name: "Tent", PriceCents: 14999, Currency: "USD", Status: StatusActive}
	new := old
	new.PriceCents = 13999

	if got, want := Diff(old, new), "price 149.99 USD -> 139.99 USD"; got != want {
		t.Errorf("Diff() = %q, want %q", got, want)
	}
	if got := Diff(old, old); got != "no field changes" {
		t.Errorf("Diff() of equal products = %q", got)
	}
}
