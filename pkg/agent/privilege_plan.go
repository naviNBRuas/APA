package agent

import (
	"os/user"
)

// PrivilegePlan describes steps to expand privileges using configured trust relationships.
type PrivilegePlan struct {
	CurrentUser string
	Groups      []string
	Suggested   []string
}

// PrivilegePlanner enumerates role assignments and suggests progressive steps.
type PrivilegePlanner struct{}

// Plan builds a privilege plan using existing OS roles (no exploit primitives).
func (PrivilegePlanner) Plan() PrivilegePlan {
	plan := PrivilegePlan{}
	u, err := user.Current()
	if err != nil {
		return plan
	}
	plan.CurrentUser = u.Username
	gids, err := u.GroupIds()
	if err != nil {
		return plan
	}
	for _, gid := range gids {
		g, err := user.LookupGroupId(gid)
		if err != nil {
			continue
		}
		plan.Groups = append(plan.Groups, g.Name)
	}
	for _, g := range plan.Groups {
		if g == "sudo" || g == "wheel" {
			plan.Suggested = append(plan.Suggested, "reuse sudo role for elevated actions")
		}
		if g == "docker" {
			plan.Suggested = append(plan.Suggested, "leverage docker group for namespaced actions")
		}
	}
	if len(plan.Suggested) == 0 {
		plan.Suggested = append(plan.Suggested, "request delegated role per policy")
	}
	return plan
}

// Execute records the plan; actual execution is delegated per operational policy.
func (p PrivilegePlanner) Execute(plan PrivilegePlan) []error {
	return nil
}
