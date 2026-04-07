package agent

import "fmt"

// PersistencePlan outlines integration points that survive restarts without introducing new binaries.
type PersistencePlan struct {
	SystemdUnit  string
	CronSpec     string
	IdentitySync string
}

// PersistencePlanner produces integration snippets for boot and scheduled execution.
type PersistencePlanner struct{}

func (PersistencePlanner) Plan(serviceName string, intervalMinutes int) PersistencePlan {
	if intervalMinutes <= 0 {
		intervalMinutes = 30
	}
	unit := fmt.Sprintf("[Unit]\nDescription=%s\n[Service]\nType=simple\nExecStart=/usr/bin/%s\n[Install]\nWantedBy=multi-user.target\n", serviceName, serviceName)
	cron := fmt.Sprintf("*/%d * * * * %s", intervalMinutes, serviceName)
	return PersistencePlan{
		SystemdUnit:  unit,
		CronSpec:     cron,
		IdentitySync: "on-boot: sync identities across peers",
	}
}
