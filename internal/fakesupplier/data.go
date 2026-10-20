package fakesupplier

import (
	"fmt"
	"strings"

	"supplier-catalog/internal/supplier"
)

// All data below is fictional: "Northwind Outdoor Supply", a made-up
// camping gear supplier.

func product(sku, name, desc string, priceCents int64, updatedAt string) supplier.Product {
	return supplier.Product{SKU: sku, Name: name, Description: desc, PriceCents: priceCents, Currency: "USD", UpdatedAt: updatedAt}
}

func withdrawal(sku, updatedAt string) supplier.Product {
	return supplier.Product{SKU: sku, Withdrawn: true, UpdatedAt: updatedAt}
}

// DemoUpdates returns the updates used in the demo. Suggested order:
//
//  1. NW-2025-09-01  initial catalog (one invalid record, one repeated record)
//  2. NW-2025-09-08  weekly changes (price changes, withdrawals, out-of-order record)
//  3. NW-2025-09-05  an OLDER update delivered late: must not overwrite newer data
//     or bring back withdrawn products
//  4. NW-2025-09-15  bigger update, good for showing outage / restart / resume
func DemoUpdates() []Update {
	const t1 = "2025-09-01T06:00:00Z"
	const t3 = "2025-09-05T06:00:00Z"

	return []Update{
		{
			ID:          "NW-2025-09-01",
			Description: "Initial full catalog",
			PublishedAt: t1,
			Batches: [][]supplier.Product{
				{
					product("NW-TENT-2P", "Trailhead 2-Person Tent", "Light 3-season tent for two.", 14999, t1),
					product("NW-TENT-4P", "Basecamp 4-Person Tent", "Roomy family tent.", 24999, t1),
					product("NW-BAG-20F", "Summit 20°F Sleeping Bag", "Down sleeping bag.", 8999, t1),
					product("NW-PAD-UL", "Ultralight Sleeping Pad", "Inflatable pad, 340 g.", 5999, t1),
				},
				{
					product("NW-STOVE-MINI", "Pocket Camp Stove", "Folding gas stove.", 3499, t1),
					product("NW-POT-SET", "Titanium Cook Set", "Pot, pan and lid.", 4999, t1),
					product("NW-LAMP-200", "Lumen 200 Headlamp", "Rechargeable headlamp.", 2499, t1),
					product("NW-FILTER-SQ", "Squeeze Water Filter", "Filters 0.1 micron.", 3999, t1),
				},
				{
					product("NW-PACK-40", "Ridge 40L Backpack", "Daypack with rain cover.", 12999, t1),
					product("NW-PACK-65", "Expedition 65L Backpack", "Multi-day backpack.", 19999, t1),
					product("NW-POLES-CF", "Carbon Trekking Poles", "Pair, adjustable.", 8999, t1),
					// Invalid: the supplier forgot the name. Becomes a problem.
					product("NW-CHAIR-LT", "", "Folding chair.", 6999, t1),
					// Repeated record (same data, same timestamp). Must not duplicate.
					product("NW-TENT-2P", "Trailhead 2-Person Tent", "Light 3-season tent for two.", 14999, t1),
				},
			},
		},
		{
			ID:          "NW-2025-09-08",
			Description: "Weekly changes",
			PublishedAt: "2025-09-08T06:00:00Z",
			Batches: [][]supplier.Product{
				{
					product("NW-TENT-2P", "Trailhead 2-Person Tent", "Light 3-season tent for two.", 13999, "2025-09-08T09:00:00Z"),
					// Newest data for the headlamp arrives FIRST...
					product("NW-LAMP-200", "Lumen 300 Headlamp", "Brighter rechargeable headlamp.", 2999, "2025-09-08T10:00:00Z"),
					// The chair now has a name: fixes the earlier problem.
					product("NW-CHAIR-LT", "Featherlite Camp Chair", "Folding chair.", 6999, "2025-09-08T09:00:00Z"),
					withdrawal("NW-STOVE-MINI", "2025-09-08T09:00:00Z"),
				},
				{
					// ...and an older headlamp record arrives later. Must be skipped.
					product("NW-LAMP-200", "Lumen 250 Headlamp", "Rechargeable headlamp.", 2699, "2025-09-08T08:00:00Z"),
					product("NW-HAMMOCK-SL", "Skyline Hammock", "Hammock with straps.", 5499, "2025-09-08T09:00:00Z"),
					// Withdrawal of a product we never had.
					withdrawal("NW-GPS-X1", "2025-09-08T09:00:00Z"),
					product("NW-BAG-20F", "Summit 20°F Sleeping Bag", "Down sleeping bag, now with compression sack.", 8999, "2025-09-08T09:00:00Z"),
				},
				{
					withdrawal("NW-FILTER-SQ", "2025-09-08T09:00:00Z"),
					product("NW-MUG-INS", "Insulated Camp Mug", "Keeps coffee hot for 4 hours.", 1999, "2025-09-08T09:00:00Z"),
					// Broken timestamp: rejected, stays as an open problem.
					{SKU: "NW-PACK-40", Name: "Ridge 40L Backpack", PriceCents: 11999, Currency: "USD", UpdatedAt: "next tuesday"},
				},
			},
		},
		{
			ID:          "NW-2025-09-05",
			Description: "Mid-week corrections (delivered late)",
			PublishedAt: t3,
			Batches: [][]supplier.Product{
				{
					// Older than the withdrawal on 09-08: must NOT bring the stove back.
					product("NW-STOVE-MINI", "Pocket Camp Stove", "Folding gas stove.", 3299, t3),
					// Older than the 09-08 price: must NOT overwrite it.
					product("NW-TENT-2P", "Trailhead 2-Person Tent", "Light 3-season tent for two.", 15999, t3),
				},
				{
					// Nothing newer exists for these, so they apply.
					product("NW-POLES-CF", "Carbon Trekking Poles", "Pair, adjustable, with carry bag.", 7999, t3),
					withdrawal("NW-PAD-UL", t3),
				},
			},
		},
		bigUpdate(),
	}
}

// bigUpdate generates 6 batches of new products for the outage/resume demo.
func bigUpdate() Update {
	const at = "2025-09-15T06:00:00Z"
	colors := []string{"Pine", "Granite", "Glacier", "Ember", "Moss", "Dusk"}
	items := []struct {
		code, name string
		price      int64
	}{
		{"FLEECE", "Fleece Jacket", 6999},
		{"BEANIE", "Wool Beanie", 1999},
		{"GLOVES", "Trail Gloves", 2999},
		{"SOCKS", "Merino Hiking Socks", 1499},
	}

	u := Update{ID: "NW-2025-09-15", Description: "Autumn collection (6 batches)", PublishedAt: at}
	for _, color := range colors {
		var batch []supplier.Product
		for _, it := range items {
			sku := fmt.Sprintf("NW-%s-%s", it.code, strings.ToUpper(color[:3]))
			name := fmt.Sprintf("%s %s", color, it.name)
			batch = append(batch, product(sku, name, "Autumn collection.", it.price, at))
		}
		u.Batches = append(u.Batches, batch)
	}
	return u
}
