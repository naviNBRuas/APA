document.addEventListener('DOMContentLoaded', () => {
    setupTabs();
    bindAuthUI();
    bindActions();
    bootstrap();
});

const state = {
    authenticated: false,
    config: null,
};

function setupTabs() {
    const tabLinks = document.querySelectorAll('nav a');
    const tabContents = document.querySelectorAll('.tab-content');
    tabLinks.forEach(link => {
        link.addEventListener('click', e => {
            e.preventDefault();
            tabLinks.forEach(tab => tab.classList.remove('active'));
            tabContents.forEach(content => content.classList.remove('active'));
            link.classList.add('active');
            const target = document.querySelector(link.getAttribute('href'));
            target?.classList.add('active');
        });
    });
}

function bindAuthUI() {
    const overlay = document.getElementById('auth-overlay');
    const loginBtn = document.getElementById('login-button');
    const logoutBtn = document.getElementById('logout-button');
    const form = document.getElementById('login-form');
    const tokenInput = document.getElementById('token-input');
    const errorEl = document.getElementById('login-error');

    const showLogin = (message = '') => {
        overlay?.classList.remove('hidden');
        if (message) errorEl.textContent = message;
        tokenInput.value = '';
        tokenInput.focus();
        updateSessionBadge(false);
    };

    const hideLogin = () => {
        overlay?.classList.add('hidden');
        errorEl.textContent = '';
    };

    loginBtn?.addEventListener('click', () => showLogin());
    logoutBtn?.addEventListener('click', async () => {
        await fetch('/auth/logout', { method: 'POST' });
        state.authenticated = false;
        updateSessionBadge(false);
        showLogin('Signed out. Re-authenticate to continue.');
    });

    form?.addEventListener('submit', async e => {
        e.preventDefault();
        const token = tokenInput.value.trim();
        if (!token) {
            errorEl.textContent = 'Token is required';
            return;
        }
        try {
            const res = await fetch('/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token }),
            });
            if (!res.ok) {
                throw new Error('Authentication failed');
            }
            hideLogin();
            state.authenticated = true;
            updateSessionBadge(true);
            await loadAllData();
        } catch (err) {
            errorEl.textContent = err.message || 'Authentication failed';
        }
    });

    // expose for other functions
    window.__showLogin = showLogin;
    window.__hideLogin = hideLogin;
}

function bindActions() {
    document.getElementById('load-module')?.addEventListener('click', async () => {
        const name = document.getElementById('module-name')?.value.trim();
        if (!name) return alert('Enter a module name');
        try {
            await api('/api/modules', { method: 'POST', body: JSON.stringify({ name }) });
            alert('Module load requested');
            await loadModules();
        } catch (err) {
            alert(err.message);
        }
    });

    document.getElementById('load-controller')?.addEventListener('click', async () => {
        const name = document.getElementById('controller-name')?.value.trim();
        if (!name) return alert('Enter a controller name');
        try {
            await api('/api/controllers', { method: 'POST', body: JSON.stringify({ name }) });
            alert('Controller load requested');
            await loadControllers();
        } catch (err) {
            alert(err.message);
        }
    });

    const settingsForm = document.getElementById('settings-form');
    settingsForm?.addEventListener('submit', async e => {
        e.preventDefault();
        if (!state.config) {
            alert('Config not yet loaded.');
            return;
        }
        const cfg = JSON.parse(JSON.stringify(state.config));
        cfg.LogLevel = document.getElementById('log-level')?.value || cfg.LogLevel;
        if (cfg.P2P) {
            const hb = Number(document.getElementById('heartbeat-interval')?.value || 0);
            if (!Number.isNaN(hb) && hb > 0) {
                cfg.P2P.HeartbeatInterval = hb * 1e9; // time.Duration in ns
            }
            const tag = document.getElementById('agent-name')?.value.trim();
            if (tag) cfg.P2P.ServiceTag = tag;
        }
        try {
            await api('/api/config', { method: 'POST', body: JSON.stringify(cfg) });
            alert('Config update requested');
            await loadConfig();
        } catch (err) {
            alert(err.message);
        }
    });
}

async function bootstrap() {
    try {
        const res = await fetch('/auth/me');
        if (res.ok) {
            state.authenticated = true;
            updateSessionBadge(true);
            await loadAllData();
            return;
        }
    } catch (err) {
        // fall-through to login overlay
    }
    updateSessionBadge(false);
    window.__showLogin?.('Authentication required');
}

async function loadAllData() {
    await Promise.allSettled([
        loadStatus(),
        loadModules(),
        loadControllers(),
        loadConfig(),
    ]);
}

async function loadStatus() {
    try {
        const status = await api('/api/status');
        document.getElementById('agent-status').textContent = 'Running';
        document.getElementById('agent-version').textContent = status.Version || '-';
        document.getElementById('agent-peer').textContent = status.PeerID || '-';
        const modules = Array.isArray(status.LoadedModules) ? status.LoadedModules.length : 0;
        document.getElementById('active-modules').textContent = modules;
        document.getElementById('module-errors').textContent = '0';
        document.getElementById('active-controllers').textContent = '-';
        document.getElementById('controller-errors').textContent = '0';
        document.getElementById('connected-peers').textContent = status.PeerCount || '-';
        document.getElementById('network-status').textContent = 'Healthy';
    } catch (err) {
        console.warn('status load failed', err);
    }
}

async function loadModules() {
    try {
        const modules = await api('/api/modules');
        const tbody = document.querySelector('#modules-table tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        modules.forEach(mod => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${mod.Name || mod.name}</td>
                <td>${mod.Version || ''}</td>
                <td>${mod.Hash ? 'Verified' : 'Loaded'}</td>
                <td>${mod.Policy || ''}</td>
            `;
            tbody.appendChild(row);
        });
        document.getElementById('active-modules').textContent = modules.length;
    } catch (err) {
        console.warn('modules load failed', err);
    }
}

async function loadControllers() {
    try {
        const controllers = await api('/api/controllers');
        const tbody = document.querySelector('#controllers-table tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        controllers.forEach(ctrl => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${ctrl.Name || ctrl.name}</td>
                <td>${ctrl.Version || ''}</td>
                <td>${ctrl.Hash ? 'Verified' : 'Loaded'}</td>
                <td>${(ctrl.Capabilities || []).join(', ')}</td>
            `;
            tbody.appendChild(row);
        });
        document.getElementById('active-controllers').textContent = controllers.length;
    } catch (err) {
        console.warn('controllers load failed', err);
    }
}

async function loadConfig() {
    try {
        const cfg = await api('/api/config');
        state.config = cfg;
        const tag = cfg.P2P?.ServiceTag || '';
        const hb = cfg.P2P?.HeartbeatInterval ? Math.round(cfg.P2P.HeartbeatInterval / 1e9) : '';
        document.getElementById('agent-name').value = tag;
        document.getElementById('log-level').value = cfg.LogLevel || 'info';
        document.getElementById('heartbeat-interval').value = hb;
    } catch (err) {
        console.warn('config load failed', err);
    }
}

async function api(path, options = {}) {
    const opts = {
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
        ...options,
    };
    const csrf = getCookie('csrf_token');
    if (opts.method && opts.method.toUpperCase() !== 'GET' && opts.method.toUpperCase() !== 'HEAD' && opts.method.toUpperCase() !== 'OPTIONS' && csrf) {
        opts.headers['X-CSRF-Token'] = csrf;
    }
    try {
        const res = await fetch(path, opts);
        if (res.status === 401) {
            state.authenticated = false;
            updateSessionBadge(false);
            window.__showLogin?.('Session expired or unauthorized');
            throw new Error('Unauthorized');
        }
        if (!res.ok) {
            const text = await res.text();
            throw new Error(text || 'Request failed');
        }
        const ct = res.headers.get('content-type') || '';
        if (ct.includes('application/json')) {
            return res.json();
        }
        return res.text();
    } catch (err) {
        throw err;
    }
}

function updateSessionBadge(ok) {
    const badge = document.getElementById('session-status');
    if (!badge) return;
    if (ok) {
        badge.textContent = 'Authenticated';
        badge.classList.remove('badge-warn');
        badge.classList.add('badge-ok');
    } else {
        badge.textContent = 'Not authenticated';
        badge.classList.remove('badge-ok');
        badge.classList.add('badge-warn');
    }
}

function getCookie(name) {
    const match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/([.$?*|{}()\[\]\\/+^])/g, '\\$1') + '=([^;]*)'));
    return match ? decodeURIComponent(match[1]) : '';
}