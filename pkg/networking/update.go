package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/naviNBRuas/APA/pkg/update"
)

// setupUpdateProtocol sets up the update protocol handler
func (p *P2P) setupUpdateProtocol() {
	p.host.SetStreamHandler(UpdateFetchProtocol, p.handleUpdateFetchRequest)
}

// handleUpdateFetchRequest handles incoming update fetch requests
func (p *P2P) handleUpdateFetchRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	decoder := json.NewDecoder(stream)
	var request struct {
		Version string `json:"version"`
	}

	if err := decoder.Decode(&request); err != nil {
		p.logger.Error("Failed to decode update fetch request", "error", err)
		return
	}

	p.mu.RLock()
	fuh := p.FetchUpdateHandler
	p.mu.RUnlock()
	if fuh != nil {
		release, data, err := fuh(request.Version)
		if err != nil {
			p.logger.Error("Failed to fetch update", "error", err)
			return
		}

		response := struct {
			Release *update.ReleaseInfo `json:"release"`
			Data    []byte              `json:"data"`
		}{
			Release: release,
			Data:    data,
		}

		encoder := json.NewEncoder(stream)
		if err := encoder.Encode(response); err != nil {
			p.logger.Error("Failed to encode update response", "error", err)
			return
		}
	} else {
		p.logger.Warn("No update handler set")
	}
}

// FetchUpdateFromPeer fetches an update from a specific peer
func (p *P2P) FetchUpdateFromPeer(ctx context.Context, peerID peer.ID, version string) (*update.ReleaseInfo, []byte, error) {
	stream, err := p.host.NewStream(ctx, peerID, UpdateFetchProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	request := struct {
		Version string `json:"version"`
	}{
		Version: version,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(request); err != nil {
		return nil, nil, fmt.Errorf("failed to encode request: %w", err)
	}

	decoder := json.NewDecoder(stream)
	var response struct {
		Release *update.ReleaseInfo `json:"release"`
		Data    []byte              `json:"data"`
	}

	if err := decoder.Decode(&response); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Release, response.Data, nil
}

// RegisterPropagationHandler registers a callback for incoming propagation payloads.
func (p *P2P) RegisterPropagationHandler(handler func(context.Context, peer.ID, PropagationPayload) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.propagationHandler = handler
}

func (p *P2P) getPropagationHandler() func(context.Context, peer.ID, PropagationPayload) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.propagationHandler
}

// SetFetchUpdateHandler sets the handler for incoming update fetch requests.
func (p *P2P) SetFetchUpdateHandler(handler func(version string) (*update.ReleaseInfo, []byte, error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.FetchUpdateHandler = handler
}
