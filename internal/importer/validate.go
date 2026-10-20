package importer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"supplier-catalog/internal/supplier"
)

// maxClockSkew is how far in the future a supplier timestamp may be.
// A timestamp far in the future would "win" against every later update and
// lock the product, so we treat it as invalid instead.
const maxClockSkew = 24 * time.Hour

// validate checks one supplier record and returns its parsed timestamp.
func validate(p supplier.Product, now time.Time) (time.Time, error) {
	if strings.TrimSpace(p.SKU) == "" {
		return time.Time{}, errors.New("missing sku")
	}
	updatedAt, err := time.Parse(time.RFC3339, p.UpdatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid updated_at %q", p.UpdatedAt)
	}
	if updatedAt.After(now.Add(maxClockSkew)) {
		return time.Time{}, fmt.Errorf("updated_at %s is in the future", p.UpdatedAt)
	}

	// A withdrawal only needs the SKU and the timestamp.
	if p.Withdrawn {
		return updatedAt, nil
	}
	if strings.TrimSpace(p.Name) == "" {
		return time.Time{}, errors.New("missing name")
	}
	if p.PriceCents <= 0 {
		return time.Time{}, fmt.Errorf("price must be positive, got %d", p.PriceCents)
	}
	if len(p.Currency) != 3 {
		return time.Time{}, fmt.Errorf("invalid currency %q", p.Currency)
	}
	return updatedAt, nil
}
