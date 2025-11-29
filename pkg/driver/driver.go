package driver

import (
	"context"
	"path/filepath"
)

// Driver defines the interface for a verified driver.
type Driver interface {
	Name() string
	Version() string
	Type() string // e.g., "network-interface", "storage-device"
	Load(ctx context.Context) error
	Unload(ctx context.Context) error
	Description() string // Human-readable description of the driver
}

// BaseDriver provides a base implementation of the Driver interface
type BaseDriver struct {
	name        string
	version     string
	driverType  string
	description string
	manager     *Manager
	path        string
}

// NewBaseDriver creates a new base driver
func NewBaseDriver(manager *Manager, name, version, driverType, description string) *BaseDriver {
	return &BaseDriver{
		name:        name,
		version:     version,
		driverType:  driverType,
		description: description,
		manager:     manager,
		path:        filepath.Join(manager.driverDir, name, name+"-"+version),
	}
}

// Name returns the driver name
func (d *BaseDriver) Name() string {
	return d.name
}

// Version returns the driver version
func (d *BaseDriver) Version() string {
	return d.version
}

// Type returns the driver type
func (d *BaseDriver) Type() string {
	return d.driverType
}

// Description returns the driver description
func (d *BaseDriver) Description() string {
	return d.description
}

// Load loads the driver
func (d *BaseDriver) Load(ctx context.Context) error {
	// Base implementation - can be overridden by specific drivers
	return nil
}

// Unload unloads the driver
func (d *BaseDriver) Unload(ctx context.Context) error {
	// Base implementation - can be overridden by specific drivers
	return nil
}