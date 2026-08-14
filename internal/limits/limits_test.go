package limits

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"xnode-agent/internal/model"
)

func TestHTTPBackendApplyAndRemove(t *testing.T) {
	var methods []string
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	b := New(srv.URL)
	if err := b.ApplyUser(context.Background(), "n 1", "in/1", "u:1", model.UserLimits{DownloadBPS: 100}); err != nil {
		t.Fatal(err)
	}
	if err := b.RemoveUser(context.Background(), "n 1", "in/1", "u:1"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != http.MethodPut || methods[1] != http.MethodDelete {
		t.Fatalf("unexpected methods: %#v", methods)
	}
	if paths[0] == paths[1] && paths[0] == "" {
		t.Fatalf("missing limiter path")
	}
}

func TestObserveOnlyRejectsStrictLimits(t *testing.T) {
	if err := (ObserveOnly{}).ApplyUser(context.Background(), "n", "i", "u", model.UserLimits{ConnectionLimit: 1}); err == nil {
		t.Fatal("expected strict limit error")
	}
	if err := (ObserveOnly{}).ApplyUser(context.Background(), "n", "i", "u", model.UserLimits{IPLimit: 1}); err != nil {
		t.Fatalf("rolling IP limit is agent-enforced and should not require strict backend: %v", err)
	}
}
