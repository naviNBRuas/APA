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
	if u, err := user.Current(); err == nil {
		plan.CurrentUser = u.Username
	}
	if u, err := user.Current(); err == nil {
		if gids, err := u.GroupIds(); err == nil {
			for _, gid := range gids {
				if g, err := user.LookupGroupId(gid); err == nil {
					plan.Groups = append(plan.Groups, g.Name)
				}
			}
		}
	}
	// Suggested steps rely on trust relationships like sudoers or service tokens.
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
