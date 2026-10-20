package catalog

import (
	"fmt"
	"strings"
	"time"
)

// Action is what happened to one incoming record.
type Action string

const (
	ActionCreate   Action = "created"
	ActionUpdate   Action = "updated"
	ActionWithdraw Action = "withdrawn"
	ActionSkip     Action = "skipped"
	ActionInvalid  Action = "invalid"
)

// Decide says what to do with an incoming record for a SKU.
//
// existing is the stored product, or nil if we have never seen the SKU.
// The supplier's updated_at decides which data is newer:
//   - incoming is older than what we have   -> skip (out of order)
//   - incoming has the same timestamp       -> skip (repeated data)
//   - incoming is newer                     -> apply it
//
// Withdrawals follow the same rule, so a withdrawn product only comes back if
// the supplier sends a record that is newer than the withdrawal.
func Decide(existing *Product, incomingUpdatedAt time.Time, withdrawn bool) (Action, string) {
	if existing == nil {
		if withdrawn {
			return ActionWithdraw, "withdrawn before we had it (kept so older data can't add it)"
		}
		return ActionCreate, "new product"
	}

	if incomingUpdatedAt.Equal(existing.SupplierUpdatedAt) {
		return ActionSkip, "already have this version"
	}
	if incomingUpdatedAt.Before(existing.SupplierUpdatedAt) {
		return ActionSkip, fmt.Sprintf("older than stored data (%s)", existing.SupplierUpdatedAt.UTC().Format(time.RFC3339))
	}

	if withdrawn {
		if existing.Status == StatusWithdrawn {
			return ActionUpdate, "still withdrawn (timestamp refreshed)"
		}
		return ActionWithdraw, "withdrawn by supplier"
	}
	if existing.Status == StatusWithdrawn {
		return ActionUpdate, "put back on sale by supplier"
	}
	return ActionUpdate, ""
}

// Diff lists the fields that differ between two versions of a product,
// e.g. "price 14999 -> 13999". It is only used for the change log.
func Diff(old, new Product) string {
	var parts []string
	if old.Name != new.Name {
		parts = append(parts, fmt.Sprintf("name %q -> %q", old.Name, new.Name))
	}
	if old.PriceCents != new.PriceCents || old.Currency != new.Currency {
		parts = append(parts, fmt.Sprintf("price %s -> %s", FormatPrice(old.PriceCents, old.Currency), FormatPrice(new.PriceCents, new.Currency)))
	}
	if old.Description != new.Description {
		parts = append(parts, "description changed")
	}
	if old.Status != new.Status {
		parts = append(parts, fmt.Sprintf("status %s -> %s", old.Status, new.Status))
	}
	if len(parts) == 0 {
		return "no field changes"
	}
	return strings.Join(parts, "; ")
}

// FormatPrice turns 1999 + "USD" into "19.99 USD".
func FormatPrice(cents int64, currency string) string {
	return fmt.Sprintf("%d.%02d %s", cents/100, cents%100, currency)
}
