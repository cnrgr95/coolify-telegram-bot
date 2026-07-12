package coolify

import (
	"encoding/json"
	"testing"
)

func TestResourceServerStatusAcceptsBooleanAndString(t *testing.T) {
	for _, test := range []struct {
		payload string
		want    string
	}{
		{`{"server_status":true}`, "true"},
		{`{"server_status":false}`, "false"},
		{`{"server_status":"running"}`, "running"},
	} {
		var resource Resource
		if err := json.Unmarshal([]byte(test.payload), &resource); err != nil {
			t.Fatalf("unmarshal %s: %v", test.payload, err)
		}
		if got := string(resource.ServerStatus); got != test.want {
			t.Fatalf("server status = %q, want %q", got, test.want)
		}
	}
}
