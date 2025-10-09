package driver

import (
	"context"
)

// Driver defines the interface for a verified driver.
type Driver interface {
	Name() string
	Version() string
	Type() string // e.g., "network-interface", "storage-device"
	Load(ctx context.Context) error
	Unload(ctx context.Context) error
}
