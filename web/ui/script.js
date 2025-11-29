// Tab navigation
document.addEventListener('DOMContentLoaded', function() {
    // Tab navigation
    const tabLinks = document.querySelectorAll('nav a');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            
            // Remove active class from all tabs
            tabLinks.forEach(tab => tab.classList.remove('active'));
            tabContents.forEach(content => content.classList.remove('active'));
            
            // Add active class to clicked tab
            this.classList.add('active');
            
            // Show corresponding content
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.classList.add('active');
            }
        });
    });
    
    // Load sample data
    loadSampleData();
    
    // Form submission
    const settingsForm = document.getElementById('settings-form');
    if (settingsForm) {
        settingsForm.addEventListener('submit', function(e) {
            e.preventDefault();
            alert('Settings saved successfully!');
        });
    }
    
    // Button event handlers
    document.getElementById('load-module')?.addEventListener('click', function() {
        alert('Load module functionality would be implemented here');
    });
    
    document.getElementById('unload-module')?.addEventListener('click', function() {
        alert('Unload module functionality would be implemented here');
    });
    
    document.getElementById('load-controller')?.addEventListener('click', function() {
        alert('Load controller functionality would be implemented here');
    });
    
    document.getElementById('unload-controller')?.addEventListener('click', function() {
        alert('Unload controller functionality would be implemented here');
    });
    
    document.getElementById('add-policy')?.addEventListener('click', function() {
        alert('Add policy functionality would be implemented here');
    });
    
    document.getElementById('edit-policy')?.addEventListener('click', function() {
        alert('Edit policy functionality would be implemented here');
    });
    
    document.getElementById('remove-policy')?.addEventListener('click', function() {
        alert('Remove policy functionality would be implemented here');
    });
});

// Load sample data for demonstration
function loadSampleData() {
    // Sample modules data
    const modules = [
        { name: 'simple-adder', version: 'v1.0.0', status: 'Active' },
        { name: 'system-info', version: 'v1.2.1', status: 'Active' },
        { name: 'data-logger', version: 'v1.0.5', status: 'Active' },
        { name: 'net-monitor', version: 'v2.0.0', status: 'Active' },
        { name: 'crypto-hasher', version: 'v1.1.0', status: 'Error' }
    ];
    
    // Populate modules table
    const modulesTable = document.querySelector('#modules-table tbody');
    if (modulesTable) {
        modulesTable.innerHTML = '';
        modules.forEach(module => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${module.name}</td>
                <td>${module.version}</td>
                <td>${module.status}</td>
                <td>
                    <button onclick="configureModule('${module.name}')">Configure</button>
                    <button onclick="unloadModule('${module.name}')">Unload</button>
                </td>
            `;
            modulesTable.appendChild(row);
        });
    }
    
    // Sample controllers data
    const controllers = [
        { name: 'task-orchestrator', version: 'v1.0.0', status: 'Active' },
        { name: 'health-controller', version: 'v1.2.1', status: 'Active' },
        { name: 'recovery-controller', version: 'v1.0.5', status: 'Active' },
        { name: 'example-controller', version: 'v1.0.0', status: 'Active' },
        { name: 'p2p-router', version: 'v1.0.0', status: 'Active' }
    ];
    
    // Populate controllers table
    const controllersTable = document.querySelector('#controllers-table tbody');
    if (controllersTable) {
        controllersTable.innerHTML = '';
        controllers.forEach(controller => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${controller.name}</td>
                <td>${controller.version}</td>
                <td>${controller.status}</td>
                <td>
                    <button onclick="configureController('${controller.name}')">Configure</button>
                    <button onclick="unloadController('${controller.name}')">Unload</button>
                </td>
            `;
            controllersTable.appendChild(row);
        });
    }
    
    // Sample policies data
    const policies = [
        { name: 'module-policy', type: 'OPA/Rego', status: 'Active' },
        { name: 'controller-policy', type: 'OPA/Rego', status: 'Active' },
        { name: 'network-policy', type: 'OPA/Rego', status: 'Active' },
        { name: 'security-policy', type: 'OPA/Rego', status: 'Active' }
    ];
    
    // Populate policies table
    const policiesTable = document.querySelector('#policies-table tbody');
    if (policiesTable) {
        policiesTable.innerHTML = '';
        policies.forEach(policy => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${policy.name}</td>
                <td>${policy.type}</td>
                <td>${policy.status}</td>
                <td>
                    <button onclick="editPolicy('${policy.name}')">Edit</button>
                    <button onclick="removePolicy('${policy.name}')">Remove</button>
                </td>
            `;
            policiesTable.appendChild(row);
        });
    }
    
    // Sample peers data
    const peers = [
        { id: 'QmPeer1', status: 'Connected', lastSeen: '2 minutes ago' },
        { id: 'QmPeer2', status: 'Connected', lastSeen: '5 minutes ago' },
        { id: 'QmPeer3', status: 'Connected', lastSeen: '1 minute ago' },
        { id: 'QmPeer4', status: 'Disconnected', lastSeen: '1 hour ago' },
        { id: 'QmPeer5', status: 'Connected', lastSeen: '30 seconds ago' }
    ];
    
    // Populate peers table
    const peersTable = document.querySelector('#peers-table tbody');
    if (peersTable) {
        peersTable.innerHTML = '';
        peers.forEach(peer => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${peer.id}</td>
                <td>${peer.status}</td>
                <td>${peer.lastSeen}</td>
                <td>
                    <button onclick="connectToPeer('${peer.id}')">Connect</button>
                    <button onclick="disconnectFromPeer('${peer.id}')">Disconnect</button>
                </td>
            `;
            peersTable.appendChild(row);
        });
    }
}

// Module functions
function configureModule(moduleName) {
    alert(`Configure module: ${moduleName}`);
}

function unloadModule(moduleName) {
    if (confirm(`Are you sure you want to unload module ${moduleName}?`)) {
        alert(`Unloading module: ${moduleName}`);
        // In a real implementation, this would make an API call to unload the module
    }
}

// Controller functions
function configureController(controllerName) {
    alert(`Configure controller: ${controllerName}`);
}

function unloadController(controllerName) {
    if (confirm(`Are you sure you want to unload controller ${controllerName}?`)) {
        alert(`Unloading controller: ${controllerName}`);
        // In a real implementation, this would make an API call to unload the controller
    }
}

// Policy functions
function editPolicy(policyName) {
    alert(`Edit policy: ${policyName}`);
}

function removePolicy(policyName) {
    if (confirm(`Are you sure you want to remove policy ${policyName}?`)) {
        alert(`Removing policy: ${policyName}`);
        // In a real implementation, this would make an API call to remove the policy
    }
}

// Peer functions
function connectToPeer(peerId) {
    alert(`Connecting to peer: ${peerId}`);
    // In a real implementation, this would make an API call to connect to the peer
}

function disconnectFromPeer(peerId) {
    if (confirm(`Are you sure you want to disconnect from peer ${peerId}?`)) {
        alert(`Disconnecting from peer: ${peerId}`);
        // In a real implementation, this would make an API call to disconnect from the peer
    }
}