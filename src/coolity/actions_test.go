package coolify

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeploymentActionsAreNeverCached(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"queued","deployment_uuid":"deployment-%d"}`, calls)
	}))
	defer server.Close()
	client := NewClient(server.URL, "token", server.Client(), time.Hour)
	first, err := client.StartApplicationDeployment("app", true, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.StartApplicationDeployment("app", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("deployment endpoint called %d times, want 2", calls)
	}
	if first.DeploymentUUID == second.DeploymentUUID {
		t.Fatal("second deployment reused cached response")
	}
}

func TestListDatabasesIncludesServiceDatabases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/databases":
			fmt.Fprint(w, `[{"uuid":"standalone","name":"PostgreSql"}]`)
		case "/api/v1/services":
			fmt.Fprint(w, `[{"databases":[{"uuid":"service-db","name":"postgres"},{"uuid":"redis","name":"redis"}]}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL, "token", server.Client(), 0)
	resources, err := client.ListDatabases()
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 3 {
		t.Fatalf("got %d databases, want 3", len(resources))
	}
}
