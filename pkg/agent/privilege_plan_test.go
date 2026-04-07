package agent

import "testing"

func TestPrivilegePlanner(t *testing.T) {
	plan := PrivilegePlanner{}.Plan()
	if plan.CurrentUser == "" {
		t.Fatalf("expected current user")
	}
	if len(plan.Suggested) == 0 {
		t.Fatalf("expected suggestions")
	}
}
