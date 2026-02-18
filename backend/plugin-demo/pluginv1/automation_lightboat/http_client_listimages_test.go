package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListImages_DoesNotEmptyWhenLineImageIDsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/api/cloud/mirror_image":
			_, _ = w.Write([]byte(`{"code":1,"msg":"succ","data":[{"id":101,"name":"ubuntu","type":"ubuntu"}]}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cli := NewClient(srv.URL+"/index.php/api/cloud", "k", 2*time.Second)
	items, err := cli.ListImages(context.Background(), 9)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 image, got %d", len(items))
	}
	if items[0].ImageID != 101 {
		t.Fatalf("unexpected image id: %d", items[0].ImageID)
	}
}
