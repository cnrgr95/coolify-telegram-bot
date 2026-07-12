package management

import (
	"context"
	"errors"
	"testing"
)

func TestExecuteRejectsUnauthorizedActor(t *testing.T) {
	service := NewService(Operations{Authorize: func(string, Action) bool { return false }})
	_, err := service.Execute(context.Background(), Request{Actor: Actor{ID: "1", Role: "viewer"}, Action: ActionRestart, ResourceID: "app-1"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestExecuteRedeployUsesForceDeployment(t *testing.T) {
	var forced bool
	service := NewService(Operations{
		Authorize: func(string, Action) bool { return true },
		Deploy: func(_ context.Context, resourceID string, force bool) (string, error) {
			forced = force
			if resourceID != "app-1" {
				t.Fatalf("unexpected resource %q", resourceID)
			}
			return "deployment-1", nil
		},
	})
	result, err := service.Execute(context.Background(), Request{Actor: Actor{ID: "1", Role: "operator"}, Action: ActionRedeploy, ResourceID: "app-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !forced || result.OperationID != "deployment-1" || !result.AuditPending {
		t.Fatalf("unexpected result: %#v, forced=%v", result, forced)
	}
}
