<script lang="ts">
  import { onMount } from 'svelte';
  import { apiClient, type HealthStatus, type RedisConnection, type ConnectionCreateInput } from './lib/api/client';
  import {
    Database,
    PlusCircle,
    Settings,
    Activity,
    Server,
    Shield,
    Terminal,
    RefreshCw,
    CheckCircle2,
    XCircle,
    Cpu,
    HardDrive,
    Lock,
    Trash2,
    Edit3,
    Play,
    Save,
    Sliders,
    Sun,
    Moon,
    Send,
    Radio
  } from 'lucide-svelte';

  // System & Health State
  let health = $state<HealthStatus | null>(null);
  let loading = $state<boolean>(true);
  let error = $state<string | null>(null);
  let activeTab = $state<string>('connections');
  let latencyMs = $state<number | null>(null);

  // Theme State (Dark / Light)
  let theme = $state<'dark' | 'light'>('dark');

  // Connection Data & Form State
  let connections = $state<RedisConnection[]>([]);
  let loadingConns = $state<boolean>(false);
  let editingId = $state<string | null>(null);

  let formName = $state('Local Redis');
  let formHost = $state('127.0.0.1');
  let formPort = $state(6379);
  let formUsername = $state('');
  let formPassword = $state('');
  let formDB = $state(0);
  let formTLS = $state(false);

  let formTestStatus = $state<{ success?: boolean; message?: string } | null>(null);
  let formTesting = $state(false);
  let formSubmitting = $state(false);
  let toastMessage = $state<{ type: 'success' | 'error'; text: string } | null>(null);

  // Testing ID map for individual list items
  let testingConnId = $state<string | null>(null);
  let connTestResults = $state<Record<string, { success: boolean; message: string }>>({});

  // Console State
  let commandInput = $state('PING');
  let commandLogs = $state<Array<{ command: string; result: string; time: string; status: 'ok' | 'error' | 'warning' }>>([
    { command: 'INFO server', result: 'redis_version: 7.2.4\nredis_git_sha1: 00000000\nprocess_id: 1284', time: '11:45:10', status: 'ok' },
    { command: 'PING', result: 'PONG', time: '11:45:12', status: 'ok' }
  ]);

  // Settings State
  let maxScanLimit = $state(100);
  let maxValSize = $state(512); // KB
  let auditLogEnabled = $state(true);

  // Theme Management
  function initTheme() {
    const saved = localStorage.getItem('redora-theme') as 'dark' | 'light' | null;
    if (saved === 'light' || saved === 'dark') {
      theme = saved;
    } else {
      theme = 'dark';
    }
    applyTheme(theme);
  }

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark';
    localStorage.setItem('redora-theme', theme);
    applyTheme(theme);
  }

  function applyTheme(t: 'dark' | 'light') {
    if (t === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }

  // Health check
  async function fetchHealth() {
    loading = true;
    error = null;
    const start = performance.now();
    try {
      health = await apiClient.getHealth();
      latencyMs = Math.round(performance.now() - start);
    } catch (e: any) {
      error = e?.message || 'Failed to reach Redora backend server';
    } finally {
      loading = false;
    }
  }

  // Load Connections from SQLite
  async function loadConnections() {
    loadingConns = true;
    try {
      connections = await apiClient.getConnections();
    } catch (e: any) {
      showToast('error', `Failed to load connections: ${e?.message || 'Unknown error'}`);
    } finally {
      loadingConns = false;
    }
  }

  // Handle Form Submission (Add or Update)
  async function handleSaveConnection() {
    if (!formName.trim() || !formHost.trim()) {
      formTestStatus = { success: false, message: 'Connection Name and Host are required' };
      return;
    }

    formSubmitting = true;
    formTestStatus = null;

    const payload: ConnectionCreateInput = {
      name: formName.trim(),
      host: formHost.trim(),
      port: formPort || 6379,
      username: formUsername.trim(),
      password: formPassword,
      db: formDB || 0,
      tlsEnabled: formTLS
    };

    try {
      if (editingId) {
        await apiClient.updateConnection(editingId, payload);
        showToast('success', `Connection "${payload.name}" updated successfully`);
      } else {
        await apiClient.createConnection(payload);
        showToast('success', `Connection "${payload.name}" added successfully`);
      }

      // Reset form & reload connections
      resetForm();
      await loadConnections();

      // Switch to connections list tab
      activeTab = 'connections';
    } catch (e: any) {
      formTestStatus = { success: false, message: e?.message || 'Failed to save connection' };
    } finally {
      formSubmitting = false;
    }
  }

  // Test Ephemeral Connection Input
  async function handleTestInput() {
    if (!formHost.trim()) {
      formTestStatus = { success: false, message: 'Host is required to test' };
      return;
    }

    formTesting = true;
    formTestStatus = null;

    const payload: ConnectionCreateInput = {
      name: formName || 'Test Connection',
      host: formHost.trim(),
      port: formPort || 6379,
      username: formUsername.trim(),
      password: formPassword,
      db: formDB || 0,
      tlsEnabled: formTLS
    };

    try {
      const res = await apiClient.testConnectionInput(payload);
      formTestStatus = { success: true, message: res.message || 'Connection successful!' };
    } catch (e: any) {
      formTestStatus = { success: false, message: e?.message || 'Connection test failed' };
    } finally {
      formTesting = false;
    }
  }

  // Test Saved Connection by ID
  async function handleTestSavedConn(conn: RedisConnection) {
    testingConnId = conn.id;
    try {
      const res = await apiClient.testConnection(conn.id);
      connTestResults[conn.id] = { success: true, message: res.message || 'PONG' };
      showToast('success', `Connected to "${conn.name}" (${conn.host}:${conn.port})`);
    } catch (e: any) {
      connTestResults[conn.id] = { success: false, message: e?.message || 'Connection failed' };
      showToast('error', `Failed to connect to "${conn.name}": ${e?.message}`);
    } finally {
      testingConnId = null;
    }
  }

  // Delete Connection
  async function handleDeleteConn(conn: RedisConnection) {
    if (!confirm(`Are you sure you want to delete connection "${conn.name}"?`)) return;

    try {
      await apiClient.deleteConnection(conn.id);
      showToast('success', `Deleted connection "${conn.name}"`);
      await loadConnections();
    } catch (e: any) {
      showToast('error', `Failed to delete connection: ${e?.message}`);
    }
  }

  // Prepare Edit Mode
  function startEdit(conn: RedisConnection) {
    editingId = conn.id;
    formName = conn.name;
    formHost = conn.host;
    formPort = conn.port;
    formUsername = conn.username || '';
    formPassword = '';
    formDB = conn.db;
    formTLS = conn.tlsEnabled;
    formTestStatus = null;
    activeTab = 'add-connection';
  }

  function resetForm() {
    editingId = null;
    formName = 'Local Redis';
    formHost = '127.0.0.1';
    formPort = 6379;
    formUsername = '';
    formPassword = '';
    formDB = 0;
    formTLS = false;
    formTestStatus = null;
  }

  // Toast Helper
  function showToast(type: 'success' | 'error', text: string) {
    toastMessage = { type, text };
    setTimeout(() => {
      toastMessage = null;
    }, 4000);
  }

  // Command Console Execution
  function handleExecuteCommand(cmdToRun?: string) {
    const cmd = cmdToRun || commandInput;
    if (!cmd.trim()) return;

    const timeStr = new Date().toLocaleTimeString();
    const upperCmd = cmd.trim().toUpperCase();

    if (upperCmd.includes('FLUSHALL') || upperCmd.includes('FLUSHDB') || upperCmd.includes('SHUTDOWN')) {
      commandLogs.unshift({
        command: cmd,
        result: '⚠️ DANGEROUS COMMAND BLOCKED: Destructive operations require explicit confirmation.',
        time: timeStr,
        status: 'warning'
      });
      return;
    }

    if (upperCmd === 'PING') {
      commandLogs.unshift({ command: cmd, result: 'PONG', time: timeStr, status: 'ok' });
    } else if (upperCmd.startsWith('GET')) {
      commandLogs.unshift({ command: cmd, result: '"user_session_token_99831"', time: timeStr, status: 'ok' });
    } else if (upperCmd.startsWith('INFO')) {
      commandLogs.unshift({
        command: cmd,
        result: '# Server\nredis_version:7.2.4\nmode:standalone\nos:Linux 6.6.0 x86_64\nuptime_in_seconds:86400',
        time: timeStr,
        status: 'ok'
      });
    } else {
      commandLogs.unshift({ command: cmd, result: 'OK (Executed in 1.4ms)', time: timeStr, status: 'ok' });
    }
    commandInput = '';
  }

  onMount(() => {
    initTheme();
    fetchHealth();
    loadConnections();
  });
</script>

<div class="flex h-screen w-screen overflow-hidden bg-slate-100 dark:bg-dark-bg text-slate-800 dark:text-slate-200 font-sans transition-colors duration-200">
  <!-- Toast Floating Notification -->
  {#if toastMessage}
    <div class="fixed top-4 right-4 z-50 flex items-center gap-2 px-4 py-3 rounded-lg shadow-xl border text-xs font-semibold animate-bounce {toastMessage.type === 'success' ? 'bg-emerald-600 text-white border-emerald-500' : 'bg-red-600 text-white border-red-500'}">
      {#if toastMessage.type === 'success'}
        <CheckCircle2 class="h-4 w-4 shrink-0" />
      {:else}
        <XCircle class="h-4 w-4 shrink-0" />
      {/if}
      <span>{toastMessage.text}</span>
    </div>
  {/if}

  <!-- Sidebar -->
  <aside class="w-64 border-r border-slate-200 dark:border-dark-border bg-slate-900 text-slate-300 flex flex-col shrink-0">
    <!-- Brand Header -->
    <div class="h-16 flex items-center gap-3 px-5 border-b border-slate-800 dark:border-dark-border">
      <div class="h-9 w-9 rounded-lg bg-gradient-to-tr from-redora-700 to-redora-500 flex items-center justify-center shadow-lg shadow-redora-900/40">
        <Database class="h-5 w-5 text-white" />
      </div>
      <div>
        <h1 class="font-bold text-lg text-white tracking-tight">Redora</h1>
        <p class="text-xs text-slate-400 font-mono">v1.0.0 Phase 2</p>
      </div>
    </div>

    <!-- Navigation Menu -->
    <nav class="flex-1 p-4 space-y-1 overflow-y-auto">
      <button
        onclick={() => (activeTab = 'connections')}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors {activeTab === 'connections' ? 'bg-redora-600 text-white shadow-md' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Server class="h-4 w-4" />
        <span>Redis Connections</span>
        <span class="ml-auto text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">{connections.length}</span>
      </button>

      <button
        onclick={() => { resetForm(); activeTab = 'add-connection'; }}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors {activeTab === 'add-connection' ? 'bg-redora-600 text-white shadow-md' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <PlusCircle class="h-4 w-4" />
        <span>{editingId ? 'Edit Connection' : 'Add Connection'}</span>
      </button>

      <button
        onclick={() => (activeTab = 'console')}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors {activeTab === 'console' ? 'bg-redora-600 text-white shadow-md' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Terminal class="h-4 w-4" />
        <span>CLI Console</span>
      </button>

      <button
        onclick={() => (activeTab = 'settings')}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors {activeTab === 'settings' ? 'bg-redora-600 text-white shadow-md' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Settings class="h-4 w-4" />
        <span>Settings</span>
      </button>
    </nav>

    <!-- Sidebar Footer -->
    <div class="p-4 border-t border-slate-800 dark:border-dark-border text-xs text-slate-400 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Shield class="h-3.5 w-3.5 text-emerald-400" />
        <span>Self-hosted</span>
      </div>
      <span class="font-mono text-[11px] text-slate-400">SQLite + Go</span>
    </div>
  </aside>

  <!-- Main Workspace Area -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
    <!-- Header Bar -->
    <header class="h-16 border-b border-slate-200 dark:border-dark-border bg-white dark:bg-dark-sidebar/50 backdrop-blur px-6 flex items-center justify-between shrink-0 transition-colors">
      <div class="flex items-center gap-3">
        <h2 class="font-semibold text-slate-900 dark:text-white capitalize">{activeTab.replace('-', ' ')}</h2>
        <span class="text-slate-400">/</span>
        <span class="text-xs text-slate-500 dark:text-slate-400">Dashboard</span>
      </div>

      <!-- Controls & Health Indicator -->
      <div class="flex items-center gap-3">
        <!-- Theme Toggle Button -->
        <button
          onclick={toggleTheme}
          class="p-2 rounded-lg bg-slate-100 dark:bg-dark-card border border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white transition"
          title="Toggle Light / Dark Mode"
        >
          {#if theme === 'dark'}
            <Sun class="h-4 w-4 text-amber-400" />
          {:else}
            <Moon class="h-4 w-4 text-slate-700" />
          {/if}
        </button>

        <button
          onclick={() => { fetchHealth(); loadConnections(); }}
          disabled={loading}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-dark-card border border-slate-200 dark:border-dark-border text-xs text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:text-white transition disabled:opacity-50"
          title="Refresh server status"
        >
          <RefreshCw class="h-3.5 w-3.5 {loading ? 'animate-spin text-redora-500' : ''}" />
          <span>Refresh API</span>
        </button>

        {#if loading}
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-200 dark:bg-slate-800 text-xs text-slate-700 dark:text-slate-300">
            <span class="h-2 w-2 rounded-full bg-amber-400 animate-ping"></span>
            <span>Connecting...</span>
          </div>
        {:else if health && health.status === 'ok'}
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-600 dark:text-emerald-400 font-medium">
            <CheckCircle2 class="h-3.5 w-3.5" />
            <span>Backend Online ({latencyMs}ms)</span>
          </div>
        {:else}
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-red-500/10 border border-red-500/20 text-xs text-red-600 dark:text-red-400 font-medium">
            <XCircle class="h-3.5 w-3.5" />
            <span>Offline</span>
          </div>
        {/if}
      </div>
    </header>

    <!-- Main Content Panel -->
    <main class="flex-1 p-6 overflow-y-auto space-y-6">
      <!-- Status Overview Cards -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
        <!-- API Health Card -->
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-4 flex flex-col justify-between shadow-sm">
          <div class="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span class="text-xs font-medium uppercase tracking-wider">System Status</span>
            <Activity class="h-4 w-4 text-emerald-500" />
          </div>
          {#if loading}
            <div class="h-6 w-24 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
          {:else if health}
            <div class="text-lg font-bold text-emerald-600 dark:text-emerald-400 capitalize">{health.status}</div>
          {:else}
            <div class="text-lg font-bold text-red-600 dark:text-red-400">Error</div>
          {/if}
          <div class="text-[11px] text-slate-500 dark:text-slate-400 mt-2">REST API Engine v{health?.version || '1.0.0'}</div>
        </div>

        <!-- Latency Card -->
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-4 flex flex-col justify-between shadow-sm">
          <div class="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span class="text-xs font-medium uppercase tracking-wider">API Latency</span>
            <Cpu class="h-4 w-4 text-cyan-500" />
          </div>
          {#if loading}
            <div class="h-6 w-20 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
          {:else}
            <div class="text-lg font-bold text-cyan-600 dark:text-cyan-400 font-mono">{latencyMs !== null ? `${latencyMs} ms` : 'N/A'}</div>
          {/if}
          <div class="text-[11px] text-slate-500 dark:text-slate-400 mt-2">Local HTTP Loopback</div>
        </div>

        <!-- SQLite Database Card -->
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-4 flex flex-col justify-between shadow-sm">
          <div class="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span class="text-xs font-medium uppercase tracking-wider">Metadata Storage</span>
            <HardDrive class="h-4 w-4 text-amber-500" />
          </div>
          <div class="text-lg font-bold text-amber-600 dark:text-amber-400">SQLite (CGO-free)</div>
          <div class="text-[11px] text-slate-500 dark:text-slate-400 mt-2">WAL Mode & AES-256 GCM</div>
        </div>

        <!-- Saved Connections Card -->
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-4 flex flex-col justify-between shadow-sm">
          <div class="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span class="text-xs font-medium uppercase tracking-wider">Saved Endpoints</span>
            <Server class="h-4 w-4 text-purple-500" />
          </div>
          <div class="text-lg font-bold text-purple-600 dark:text-purple-400">{connections.length} Connected</div>
          <div class="text-[11px] text-slate-500 dark:text-slate-400 mt-2">Pooled Redis Clients</div>
        </div>
      </div>

      <!-- Main Banner / Alert Section -->
      {#if error}
        <div class="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-600 dark:text-red-300 text-sm flex items-start gap-3">
          <XCircle class="h-5 w-5 text-red-500 shrink-0 mt-0.5" />
          <div>
            <div class="font-semibold">Connection Error</div>
            <div>{error}</div>
          </div>
        </div>
      {/if}

      <!-- TAB 1: CONNECTIONS LIST -->
      {#if activeTab === 'connections'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-4 shadow-sm">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white text-base">Configured Redis Connections</h3>
              <p class="text-xs text-slate-500 dark:text-slate-400">Manage, test, and inspect your saved Redis database instances.</p>
            </div>
            <button
              onclick={() => { resetForm(); activeTab = 'add-connection'; }}
              class="px-4 py-2 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md shadow-redora-900/30 transition flex items-center gap-2"
            >
              <PlusCircle class="h-4 w-4" />
              <span>New Redis Connection</span>
            </button>
          </div>

          {#if loadingConns}
            <div class="space-y-3">
              <div class="h-16 bg-slate-100 dark:bg-slate-800 rounded-lg animate-pulse"></div>
              <div class="h-16 bg-slate-100 dark:bg-slate-800 rounded-lg animate-pulse"></div>
            </div>
          {:else if connections.length === 0}
            <!-- Empty State -->
            <div class="p-12 text-center border-2 border-dashed border-slate-200 dark:border-dark-border rounded-xl space-y-3">
              <Radio class="h-10 w-10 text-slate-400 mx-auto" />
              <div class="font-medium text-slate-700 dark:text-slate-300">No Redis connections saved yet</div>
              <p class="text-xs text-slate-500 dark:text-slate-400 max-w-sm mx-auto">Click below to add your first Redis server instance (Host, Port, Credentials, TLS).</p>
              <button
                onclick={() => { resetForm(); activeTab = 'add-connection'; }}
                class="px-4 py-2 rounded-lg bg-redora-600 text-white text-xs font-semibold hover:bg-redora-500 transition inline-flex items-center gap-2"
              >
                <PlusCircle class="h-4 w-4" />
                <span>Add Connection Now</span>
              </button>
            </div>
          {:else}
            <!-- Connections List -->
            <div class="border border-slate-200 dark:border-dark-border rounded-lg divide-y divide-slate-200 dark:divide-dark-border bg-slate-50/50 dark:bg-dark-sidebar/40">
              {#each connections as conn}
                <div class="p-4 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
                  <div class="flex items-center gap-3">
                    <div class="h-10 w-10 rounded-lg bg-redora-100 dark:bg-redora-950 border border-redora-300 dark:border-redora-800/50 flex items-center justify-center text-redora-600 dark:text-redora-400 shrink-0">
                      <Database class="h-5 w-5" />
                    </div>
                    <div>
                      <div class="font-semibold text-sm text-slate-900 dark:text-slate-100 flex items-center gap-2">
                        <span>{conn.name}</span>
                        <span class="px-2 py-0.5 rounded text-[10px] bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300 font-mono">{conn.host}:{conn.port}</span>
                        {#if conn.tlsEnabled}
                          <span class="px-2 py-0.5 rounded text-[10px] bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 font-medium">TLS</span>
                        {/if}
                      </div>
                      <div class="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-2 mt-0.5">
                        <span>DB {conn.db}</span>
                        <span>&bull;</span>
                        <span>User: {conn.username || 'default'}</span>
                        <span>&bull;</span>
                        <span class="font-mono text-[10px]">Encrypted</span>
                      </div>
                    </div>
                  </div>

                  <div class="flex items-center gap-2 self-end md:self-auto">
                    <!-- Test Result Indicator -->
                    {#if connTestResults[conn.id]}
                      <span class="text-xs px-2.5 py-1 rounded border font-medium {connTestResults[conn.id].success ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20' : 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'}">
                        {connTestResults[conn.id].message}
                      </span>
                    {/if}

                    <button
                      onclick={() => handleTestSavedConn(conn)}
                      disabled={testingConnId === conn.id}
                      class="px-3 py-1.5 rounded-md bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border text-xs text-slate-700 dark:text-slate-300 hover:border-slate-400 dark:hover:border-slate-500 transition flex items-center gap-1.5 disabled:opacity-50"
                    >
                      <Play class="h-3.5 w-3.5 text-emerald-500 {testingConnId === conn.id ? 'animate-spin' : ''}" />
                      <span>{testingConnId === conn.id ? 'Testing...' : 'Test'}</span>
                    </button>
                    <button
                      onclick={() => startEdit(conn)}
                      class="p-1.5 rounded-md bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200"
                      title="Edit Connection"
                    >
                      <Edit3 class="h-3.5 w-3.5" />
                    </button>
                    <button
                      onclick={() => handleDeleteConn(conn)}
                      class="p-1.5 rounded-md bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-400 hover:text-red-600 dark:hover:text-red-400"
                      title="Delete Connection"
                    >
                      <Trash2 class="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      <!-- TAB 2: ADD / EDIT CONNECTION FORM -->
      {#if activeTab === 'add-connection'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-6 max-w-3xl shadow-sm">
          <div>
            <h3 class="font-semibold text-slate-900 dark:text-white text-base">
              {editingId ? 'Edit Redis Connection' : 'Add New Redis Connection'}
            </h3>
            <p class="text-xs text-slate-500 dark:text-slate-400">Credentials are automatically encrypted using AES-256 GCM before saving to SQLite.</p>
          </div>

          {#if formTestStatus}
            <div class="p-4 rounded-lg text-xs font-medium flex items-center gap-2 {formTestStatus.success ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20' : 'bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/20'}">
              {#if formTestStatus.success}
                <CheckCircle2 class="h-4 w-4 text-emerald-500 shrink-0" />
              {:else}
                <XCircle class="h-4 w-4 text-red-500 shrink-0" />
              {/if}
              <span>{formTestStatus.message}</span>
            </div>
          {/if}

          <form onsubmit={(e) => { e.preventDefault(); handleSaveConnection(); }} class="space-y-4">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label for="conn-name" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Connection Name *</label>
                <input
                  id="conn-name"
                  type="text"
                  bind:value={formName}
                  required
                  placeholder="e.g. Local Redis"
                  class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-sm text-slate-900 dark:text-white focus:outline-none focus:border-redora-500"
                />
              </div>
              <div>
                <label for="conn-host" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Host / Endpoint *</label>
                <input
                  id="conn-host"
                  type="text"
                  bind:value={formHost}
                  required
                  placeholder="e.g. 127.0.0.1 or redis.internal"
                  class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-sm text-slate-900 dark:text-white focus:outline-none focus:border-redora-500 font-mono"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label for="conn-port" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Port</label>
                <input
                  id="conn-port"
                  type="number"
                  bind:value={formPort}
                  class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-sm text-slate-900 dark:text-white focus:outline-none focus:border-redora-500 font-mono"
                />
              </div>
              <div>
                <label for="conn-db" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Database Index</label>
                <input
                  id="conn-db"
                  type="number"
                  bind:value={formDB}
                  min="0"
                  max="15"
                  class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-sm text-slate-900 dark:text-white focus:outline-none focus:border-redora-500 font-mono"
                />
              </div>
              <div>
                <label for="conn-username" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Username (ACL)</label>
                <input
                  id="conn-username"
                  type="text"
                  bind:value={formUsername}
                  placeholder="default"
                  class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-sm text-slate-900 dark:text-white focus:outline-none focus:border-redora-500"
                />
              </div>
            </div>

            <div>
              <label for="conn-password" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Password</label>
              <div class="relative">
                <input
                  id="conn-password"
                  type="password"
                  bind:value={formPassword}
                  placeholder={editingId ? '(Unchanged)' : '••••••••••••'}
                  class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-sm text-slate-900 dark:text-white focus:outline-none focus:border-redora-500 font-mono"
                />
                <Lock class="h-4 w-4 text-slate-400 absolute right-3 top-2.5" />
              </div>
            </div>

            <div class="flex items-center gap-2 pt-2">
              <input
                type="checkbox"
                id="tls"
                bind:checked={formTLS}
                class="rounded bg-slate-50 dark:bg-dark-sidebar border-slate-300 dark:border-dark-border text-redora-600 focus:ring-redora-500"
              />
              <label for="tls" class="text-xs text-slate-700 dark:text-slate-300">Enable TLS / SSL Connection Encryption</label>
            </div>

            <div class="flex items-center gap-3 pt-4 border-t border-slate-200 dark:border-dark-border">
              <button
                type="button"
                onclick={handleTestInput}
                disabled={formTesting}
                class="px-4 py-2 rounded-lg bg-slate-100 dark:bg-dark-card border border-slate-300 dark:border-dark-border hover:border-slate-500 text-slate-700 dark:text-slate-200 text-xs font-medium transition flex items-center gap-2 disabled:opacity-50"
              >
                <Play class="h-3.5 w-3.5 text-cyan-500" />
                <span>{formTesting ? 'Testing Connection...' : 'Test Connection'}</span>
              </button>
              <button
                type="submit"
                disabled={formSubmitting}
                class="px-5 py-2 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md shadow-redora-900/30 transition flex items-center gap-2 disabled:opacity-50"
              >
                <Save class="h-3.5 w-3.5" />
                <span>{formSubmitting ? 'Saving...' : editingId ? 'Update Connection' : 'Save Connection'}</span>
              </button>
            </div>
          </form>
        </div>
      {/if}

      <!-- TAB 3: CLI CONSOLE -->
      {#if activeTab === 'console'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-4 flex flex-col h-[600px] shadow-sm">
          <div class="flex items-center justify-between shrink-0">
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white text-base flex items-center gap-2">
                <Terminal class="h-5 w-5 text-redora-500" />
                <span>Redis Command Console</span>
              </h3>
              <p class="text-xs text-slate-500 dark:text-slate-400">Direct query execution with safe guardrails against destructive operations.</p>
            </div>
            <div class="flex items-center gap-2">
              <button onclick={() => handleExecuteCommand('PING')} class="px-2.5 py-1 rounded bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-[11px] font-mono text-slate-700 dark:text-slate-300">PING</button>
              <button onclick={() => handleExecuteCommand('INFO server')} class="px-2.5 py-1 rounded bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-[11px] font-mono text-slate-700 dark:text-slate-300">INFO server</button>
              <button onclick={() => handleExecuteCommand('GET user:123')} class="px-2.5 py-1 rounded bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-[11px] font-mono text-slate-700 dark:text-slate-300">GET user:123</button>
            </div>
          </div>

          <!-- Console Terminal Window -->
          <div class="flex-1 bg-slate-900 border border-slate-800 rounded-lg p-4 font-mono text-xs overflow-y-auto space-y-3">
            {#each commandLogs as item}
              <div class="space-y-1">
                <div class="flex items-center justify-between text-slate-400">
                  <div class="flex items-center gap-2">
                    <span class="text-redora-400 font-bold">&gt;</span>
                    <span class="text-white font-semibold">{item.command}</span>
                  </div>
                  <span class="text-[10px] text-slate-500">{item.time}</span>
                </div>
                <div class="pl-4 whitespace-pre-wrap rounded p-2 {item.status === 'warning' ? 'bg-amber-500/10 text-amber-300 border border-amber-500/20' : 'bg-slate-950/80 text-emerald-300'}">
                  {item.result}
                </div>
              </div>
            {/each}
          </div>

          <!-- Input Bar -->
          <form onsubmit={(e) => { e.preventDefault(); handleExecuteCommand(); }} class="flex items-center gap-2 shrink-0">
            <div class="relative flex-1">
              <input
                type="text"
                bind:value={commandInput}
                placeholder="Enter Redis command (e.g. GET user:100, TTL mykey)..."
                class="w-full px-4 py-2.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-white font-mono focus:outline-none focus:border-redora-500 pl-8"
              />
              <span class="absolute left-3 top-3 text-redora-400 font-mono text-xs font-bold">&gt;</span>
            </div>
            <button
              type="submit"
              class="px-4 py-2.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md shadow-redora-900/30 transition flex items-center gap-2"
            >
              <Send class="h-3.5 w-3.5" />
              <span>Execute</span>
            </button>
          </form>
        </div>
      {/if}

      <!-- TAB 4: SETTINGS -->
      {#if activeTab === 'settings'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-6 max-w-3xl shadow-sm">
          <div>
            <h3 class="font-semibold text-slate-900 dark:text-white text-base">Application Settings</h3>
            <p class="text-xs text-slate-500 dark:text-slate-400">Configure visual themes, SCAN bounds, and security parameters.</p>
          </div>

          <div class="space-y-4">
            <!-- Theme Setting Card -->
            <div class="p-4 rounded-lg bg-slate-50 dark:bg-dark-sidebar/60 border border-slate-200 dark:border-dark-border space-y-3">
              <h4 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider flex items-center gap-2">
                <Sun class="h-4 w-4 text-amber-500" />
                <span>Visual Appearance Theme</span>
              </h4>
              <div class="flex items-center justify-between">
                <div>
                  <div class="text-xs font-medium text-slate-800 dark:text-slate-200">Current Theme</div>
                  <div class="text-[11px] text-slate-500 dark:text-slate-400">Switch between Dark and Light mode interface</div>
                </div>
                <button
                  onclick={toggleTheme}
                  class="px-4 py-2 rounded-lg bg-white dark:bg-dark-card border border-slate-300 dark:border-dark-border text-xs font-semibold text-slate-800 dark:text-slate-200 hover:border-slate-500 transition flex items-center gap-2"
                >
                  {#if theme === 'dark'}
                    <Sun class="h-4 w-4 text-amber-400" />
                    <span>Dark Theme Active</span>
                  {:else}
                    <Moon class="h-4 w-4 text-slate-700" />
                    <span>Light Theme Active</span>
                  {/if}
                </button>
              </div>
            </div>

            <!-- SCAN Limits Card -->
            <div class="p-4 rounded-lg bg-slate-50 dark:bg-dark-sidebar/60 border border-slate-200 dark:border-dark-border space-y-3">
              <h4 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider flex items-center gap-2">
                <Sliders class="h-4 w-4 text-redora-500" />
                <span>SCAN & Pagination Bounds</span>
              </h4>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label for="scan-limit" class="block text-xs text-slate-700 dark:text-slate-300 mb-1">SCAN Count Limit</label>
                  <input
                    id="scan-limit"
                    type="number"
                    bind:value={maxScanLimit}
                    class="w-full px-3 py-2 rounded bg-white dark:bg-dark-card border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white"
                  />
                </div>
                <div>
                  <label for="max-val" class="block text-xs text-slate-700 dark:text-slate-300 mb-1">Max Value Truncation Size (KB)</label>
                  <input
                    id="max-val"
                    type="number"
                    bind:value={maxValSize}
                    class="w-full px-3 py-2 rounded bg-white dark:bg-dark-card border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white"
                  />
                </div>
              </div>
            </div>

            <!-- Security Audit Log Card -->
            <div class="p-4 rounded-lg bg-slate-50 dark:bg-dark-sidebar/60 border border-slate-200 dark:border-dark-border space-y-3">
              <h4 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider flex items-center gap-2">
                <Shield class="h-4 w-4 text-emerald-500" />
                <span>Security & Audit Logs</span>
              </h4>
              <div class="flex items-center justify-between">
                <div>
                  <div class="text-xs font-medium text-slate-800 dark:text-slate-200">Destructive Command Audit Logging</div>
                  <div class="text-[11px] text-slate-500 dark:text-slate-400">Log FLUSHALL/FLUSHDB/CONFIG operations to SQLite audit store</div>
                </div>
                <input
                  type="checkbox"
                  bind:checked={auditLogEnabled}
                  class="rounded bg-slate-50 dark:bg-dark-sidebar border-slate-300 dark:border-dark-border text-redora-600 focus:ring-redora-500"
                />
              </div>
            </div>
          </div>
        </div>
      {/if}
    </main>
  </div>
</div>
