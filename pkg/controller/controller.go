package controller

import (
	"context"
)

// Controller defines the interface for a decentralized controller module.
type Controller interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
