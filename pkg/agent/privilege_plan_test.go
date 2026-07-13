package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrivilegePlanner(t *testing.T) {
	plan := PrivilegePlanner{}.Plan()
	require.NotEmpty(t, plan.CurrentUser, "expected current user")
	require.NotEmpty(t, plan.Suggested, "expected suggestions")
}
