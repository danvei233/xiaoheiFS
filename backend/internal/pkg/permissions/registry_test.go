package permissions

import (
	"testing"

	"xiaoheiplay/internal/domain"
)

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

// TestRegionActiveMatchesRouteInference guards the region.set_active vs
// region.active inconsistency: the production source of truth is the
// route-inferred code, so the static registry must not diverge.
func TestRegionActiveMatchesRouteInference(t *testing.T) {
	code, ok := InferPermissionCode("PATCH", "/admin/api/v1/regions/:id/active")
	if !ok || code != "region.active" {
		t.Fatalf("InferPermissionCode(PATCH /regions/:id/active) = %v %q, want region.active", ok, code)
	}
	InitRegistry()
	for _, d := range GetDefinitions() {
		if d.Code == "region.set_active" {
			t.Fatalf("static registry must use region.active (route inference), found region.set_active")
		}
	}
}

// TestGetDefinitionsReturnsCopy ensures callers cannot mutate the global store
// through the returned slice.
func TestGetDefinitionsReturnsCopy(t *testing.T) {
	SetDefinitions([]domain.PermissionDefinition{{Code: "keep.me"}})
	defs := GetDefinitions()
	if len(defs) != 1 {
		t.Fatalf("unexpected definitions: %v", defs)
	}
	defs[0].Code = "mutated"
	if got := GetDefinitions()[0].Code; got != "keep.me" {
		t.Fatalf("GetDefinitions must return a copy, got %q", got)
	}
}
