package config

import (
	"path/filepath"
	"testing"

	"coolifymanager/src/database"
)

func TestPermissionMatrix(t *testing.T) {
	t.Setenv("DATA_PATH", filepath.Join(t.TempDir(), "permissions.json"))
	if err := database.Connect(""); err != nil {
		t.Fatal(err)
	}
	devIDs = []int64{1}
	if err := database.AddAuthorizedUser(2, "viewer"); err != nil {
		t.Fatal(err)
	}
	if err := database.AddAuthorizedUser(3, "operator"); err != nil {
		t.Fatal(err)
	}
	if err := database.AddAuthorizedUser(4, "admin"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		user   int64
		action string
		want   bool
	}{
		{2, "view", true}, {2, "logs", true}, {2, "restart", false}, {2, "stop", false}, {2, "delete", false}, {2, "users", false},
		{3, "view", true}, {3, "restart", true}, {3, "stop", true}, {3, "delete", false}, {3, "users", false},
		{4, "delete", true}, {4, "users", true}, {1, "delete", true},
	}
	for _, test := range tests {
		if got := Can(test.user, test.action); got != test.want {
			t.Errorf("Can(%d, %q) = %v, want %v", test.user, test.action, got, test.want)
		}
	}
}
