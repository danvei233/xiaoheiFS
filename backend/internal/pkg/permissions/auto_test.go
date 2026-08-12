package permissions

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInferPermissionCode(t *testing.T) {
	code, ok := InferPermissionCode("GET", "/admin/api/v1/users")
	if !ok || code != "user.list" {
		t.Fatalf("unexpected code: %v %s", ok, code)
	}
	code, ok = InferPermissionCode("PATCH", "/admin/api/v1/users/:id/status")
	if !ok || code != "user.update" {
		t.Fatalf("unexpected status code: %v %s", ok, code)
	}
	if _, ok := InferPermissionCode("GET", "/api/v1/users"); ok {
		t.Fatalf("expected non-admin route to be ignored")
	}
}

func TestInferPermissionCodeCMS(t *testing.T) {
	cases := []struct {
		method, path, want string
	}{
		{"GET", "/admin/api/v1/cms/posts", "cms_post.list"},
		{"POST", "/admin/api/v1/cms/posts", "cms_post.create"},
		{"GET", "/admin/api/v1/cms/posts/:id", "cms_post.view"},
		{"PATCH", "/admin/api/v1/cms/categories/:id", "cms_category.update"},
		{"DELETE", "/admin/api/v1/cms/blocks/:id", "cms_block.delete"},
		{"GET", "/admin/api/v1/cms/blocks", "cms_block.list"},
	}
	for _, tc := range cases {
		code, ok := InferPermissionCode(tc.method, tc.path)
		if !ok || code != tc.want {
			t.Fatalf("InferPermissionCode(%s %s) = %q ok=%v, want %q", tc.method, tc.path, code, ok, tc.want)
		}
	}
}

func TestBuildFromRoutes(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: "GET", Path: "/admin/api/v1/users"},
		{Method: "PATCH", Path: "/admin/api/v1/users/:id/status"},
		{Method: "GET", Path: "/admin/api/v1/orders/:id"},
	}
	defs := BuildFromRoutes(routes)
	if len(defs) == 0 {
		t.Fatalf("expected definitions")
	}
}

func TestRegistryHelpers(t *testing.T) {
	SetDefinitions(nil)
	RegisterWithFriendlyName("order.view", "Order View", "View Order", "order", 1)
	RegisterWithParent("order.update", "Order Update", "order", "order.view", 2)
	defs := GetDefinitions()
	if len(defs) < 2 {
		t.Fatalf("expected definitions")
	}
}
