package categories

import "testing"

func TestAllReturns11Categories(t *testing.T) {
	got := All()
	if len(got) != 11 {
		t.Fatalf("All() length = %d, want 11", len(got))
	}
}

func TestAllReturnsCopy(t *testing.T) {
	a := All()
	a[0].Label = "Mutated"
	b := All()
	if b[0].Label == "Mutated" {
		t.Fatal("All() returned a slice that shares state with the package-internal list")
	}
}

func TestAllLabelsAndSlugs(t *testing.T) {
	want := []Category{
		{Label: "Groceries", Slug: "groceries"},
		{Label: "Eating Out", Slug: "eating"},
		{Label: "Transport", Slug: "transport"},
		{Label: "Housing", Slug: "housing"},
		{Label: "Utilities", Slug: "utilities"},
		{Label: "Sport", Slug: "fitness"},
		{Label: "Health", Slug: "health"},
		{Label: "Entertainment", Slug: "other"},
		{Label: "Travel", Slug: "travel"},
		{Label: "Gifts", Slug: "gifts"},
		{Label: "Other", Slug: "other"},
	}
	got := All()
	for i, w := range want {
		if got[i] != w {
			t.Errorf("All()[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}
