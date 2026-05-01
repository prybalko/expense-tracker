// Package categories defines the canonical list of expense categories used
// across the API. The Label is the persisted value stored in the database;
// the Slug is a stable identifier used by the frontend to map a category to
// its glyph and color.
package categories

// Category is a single expense category.
type Category struct {
	Label string `json:"label"`
	Slug  string `json:"slug"`
}

var all = []Category{
	{Label: "Groceries", Slug: "groceries"},
	{Label: "Eating Out", Slug: "eating"},
	{Label: "Transport", Slug: "transport"},
	{Label: "Housing", Slug: "housing"},
	{Label: "Utilities", Slug: "utilities"},
	{Label: "Sport", Slug: "fitness"},
	{Label: "Health", Slug: "health"},
	{Label: "Entertainment", Slug: "entertainment"},
	{Label: "Travel", Slug: "travel"},
	{Label: "Gifts", Slug: "gifts"},
	{Label: "Other", Slug: "other"},
}

// All returns the canonical list of categories in display order.
func All() []Category {
	out := make([]Category, len(all))
	copy(out, all)
	return out
}
