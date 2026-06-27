package networking

import (
	"context"
	"fmt"
)

// PutDHTValue stores a key/value pair in the DHT for integration harnessing.
func (p *P2P) PutDHTValue(ctx context.Context, key string, val []byte) error {
	if p == nil || p.dht == nil {
		return fmt.Errorf("dht not initialized")
	}
	return p.dht.PutValue(ctx, key, val)
}

// GetDHTValue retrieves a value from the DHT.
func (p *P2P) GetDHTValue(ctx context.Context, key string) ([]byte, error) {
	if p == nil || p.dht == nil {
		return nil, fmt.Errorf("dht not initialized")
	}
	return p.dht.GetValue(ctx, key)
}
