package permissions

import "testing"

func TestRegistryInitAndLookup(t *testing.T) {
	InitRegistry()
	defs := GetDefinitions()
	if len(defs) == 0 {
		t.Fatalf("expected definitions")
	}
	cats := GetCategories()
	if len(cats) == 0 {
		t.Fatalf("expected categories")
	}
	if got := GetByCategory(defs[0].Category); len(got) == 0 {
		t.Fatalf("expected category lookup")
	}
	SetDefinitions(nil)
	if len(GetDefinitions()) != 0 {
		t.Fatalf("expected cleared definitions")
	}
}
