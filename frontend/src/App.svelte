<script lang="ts">
  import { onMount } from 'svelte';
  import { apiClient, type HealthStatus, type RedisConnection, type ConnectionCreateInput } from './lib/api/client';
  import KeyBrowser from './lib/components/KeyBrowser.svelte';
  import ValueViewer from './lib/components/ValueViewer.svelte';
  import PubSubInspector from './lib/components/PubSubInspector.svelte';
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
    Radio,
    Key,
    AlertTriangle,
    X
  } from 'lucide-svelte';

  // System & Health State
  let health = $state<HealthStatus | null>(null);
  let loading = $state<boolean>(true);
  let error = $state<string | null>(null);
  let activeTab = $state<string>('key-browser');
  let latencyMs = $state<number | null>(null);

  // Theme State
  let theme = $state<'dark' | 'light'>('dark');

  // Connections State (Unified active connection selection)
  let connections = $state<RedisConnection[]>([]);
  let activeConnId = $state<string | null>(null);
  let loadingConns = $state<boolean>(false);
  let editingId = $state<string | null>(null);

  // Key Selection State
  let selectedKey = $state<string | null>(null);

  // Connection Form State
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

  let testingConnId = $state<string | null>(null);
  let connTestResults = $state<Record<string, { success: boolean; message: string }>>({});

  // CLI Console State
  let commandInput = $state('PING');
  let commandLogs = $state<Array<{ command: string; result: string; time: string; status: 'ok' | 'error' | 'warning' }>>([]);
  let isExecutingCmd = $state<boolean>(false);

  // Dangerous Command Confirmation Modal
  let showDangerousModal = $state<boolean>(false);
  let pendingDangerousCmd = $state<string>('');

  // Settings State
  let maxScanLimit = $state(100);
  let maxValSize = $state(512);

  // Active Selected Connection Object (Unified single source of truth)
  let activeConn = $derived<RedisConnection | null>(
    connections.find((c) => c.id === activeConnId) || connections[0] || null
  );

  // Reset selected key when switching active connection
  $effect(() => {
    if (activeConnId) {
      selectedKey = null;
    }
  });

  // Theme Helpers
  function initTheme() {
    const saved = localStorage.getItem('redora-theme') as 'dark' | 'light' | null;
    theme = saved === 'light' || saved === 'dark' ? saved : 'dark';
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

  // Health
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

  // Load Connections
  async function loadConnections() {
    loadingConns = true;
    try {
      connections = await apiClient.getConnections();
      if (connections.length > 0 && !activeConnId) {
        activeConnId = connections[0].id;
      }
    } catch (e: any) {
      showToast('error', `Failed to load connections: ${e?.message}`);
    } finally {
      loadingConns = false;
    }
  }

  // Save Connection Form
  async function handleSaveConnection() {
    if (!formName.trim() || !formHost.trim()) {
      formTestStatus = { success: false, message: 'Connection Name and Host are required' };
      return;
    }

    formSubmitting = true;
    formTestStatus = null;

    const payload: Record<string, any> = {
      name: formName.trim(),
      host: formHost.trim(),
      port: formPort || 6379,
      username: formUsername.trim(),
      db: formDB || 0,
      tlsEnabled: formTLS,
    };

    if (formPassword !== '') {
      payload.password = formPassword;
    } else if (!editingId) {
      payload.password = '';
    }

    try {
      if (editingId) {
        await apiClient.updateConnection(editingId, payload);
        showToast('success', `Connection "${payload.name}" updated successfully`);
      } else {
        const newConn = await apiClient.createConnection(payload as ConnectionCreateInput);
        activeConnId = newConn.id;
        showToast('success', `Connection "${payload.name}" added successfully`);
      }

      resetForm();
      await loadConnections();
      activeTab = 'key-browser';
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
      id: editingId || undefined,
      name: formName || 'Test Connection',
      host: formHost.trim(),
      port: formPort || 6379,
      username: formUsername.trim(),
      password: formPassword,
      db: formDB || 0,
      tlsEnabled: formTLS,
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
      if (activeConnId === conn.id) {
        activeConnId = null;
      }
      await loadConnections();
    } catch (e: any) {
      showToast('error', `Failed to delete connection: ${e?.message}`);
    }
  }

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

  function showToast(type: 'success' | 'error', text: string) {
    toastMessage = { type, text };
    setTimeout(() => {
      toastMessage = null;
    }, 4000);
  }

  // CLI Command Execution (Using unified activeConnId from sidebar)
  async function handleExecuteCommand(cmdToRun?: string, force = false) {
    const cmd = cmdToRun || commandInput;
    if (!cmd.trim()) return;

    if (!activeConnId) {
      showToast('error', 'Please select or add a Redis connection first');
      return;
    }

    const upperCmd = cmd.trim().toUpperCase();

    if (!force && (upperCmd.includes('FLUSHALL') || upperCmd.includes('FLUSHDB') || upperCmd.includes('SHUTDOWN') || upperCmd.includes('CONFIG'))) {
      pendingDangerousCmd = cmd;
      showDangerousModal = true;
      return;
    }

    isExecutingCmd = true;
    const timeStr = new Date().toLocaleTimeString();

    try {
      const res = await apiClient.executeCommand(activeConnId, cmd);
      commandLogs.unshift({
        command: cmd,
        result: res.result,
        time: timeStr,
        status: res.status === 'ok' ? 'ok' : 'error',
      });
      commandInput = '';
    } catch (e: any) {
      commandLogs.unshift({
        command: cmd,
        result: `ERR: ${e?.message || 'Command execution failed'}`,
        time: timeStr,
        status: 'error',
      });
    } finally {
      isExecutingCmd = false;
      showDangerousModal = false;
    }
  }

  onMount(() => {
    initTheme();
    fetchHealth();
    loadConnections();
  });
</script>

<div class="flex h-screen w-screen overflow-hidden bg-slate-100 dark:bg-dark-bg text-slate-800 dark:text-slate-200 font-sans transition-colors duration-200">
  <!-- Toast Notification -->
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
    <div class="h-16 flex items-center gap-3 px-5 border-b border-slate-800 dark:border-dark-border shrink-0">
      <div class="h-9 w-9 rounded-lg bg-gradient-to-tr from-redora-700 to-redora-500 flex items-center justify-center shadow-lg shadow-redora-900/40 shrink-0">
        <Database class="h-5 w-5 text-white" />
      </div>
      <div class="min-w-0 flex-1">
        <h1 class="font-bold text-lg text-white tracking-tight leading-none">Redora</h1>
        <p class="text-[11px] text-slate-400 font-mono mt-1">v1.0.0</p>
      </div>
    </div>

    <!-- Unified Active Connection Selector Dropdown in Sidebar -->
    {#if connections.length > 0}
      <div class="p-3 border-b border-slate-800 shrink-0">
        <label for="sidebar-conn-select" class="block text-[10px] uppercase font-bold text-slate-400 mb-1">Target Connection</label>
        <select
          id="sidebar-conn-select"
          bind:value={activeConnId}
          class="w-full px-2.5 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-xs text-white font-medium focus:outline-none focus:border-redora-500 truncate"
        >
          {#each connections as conn}
            <option value={conn.id}>{conn.name} ({conn.host}:{conn.port})</option>
          {/each}
        </select>
      </div>
    {/if}

    <!-- Navigation Menu -->
    <nav class="flex-1 p-3 space-y-1 overflow-y-auto">
      <button
        onclick={() => (activeTab = 'key-browser')}
        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-semibold transition-colors {activeTab === 'key-browser' ? 'bg-redora-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Key class="h-4 w-4 shrink-0" />
        <span class="truncate text-left flex-1">Key Browser</span>
      </button>

      <button
        onclick={() => (activeTab = 'connections')}
        class="w-full flex items-center justify-between gap-2 px-3 py-2 rounded-lg text-xs font-semibold transition-colors {activeTab === 'connections' ? 'bg-redora-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <div class="flex items-center gap-2.5 min-w-0 flex-1">
          <Server class="h-4 w-4 shrink-0" />
          <span class="truncate text-left">Connections</span>
        </div>
        <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 shrink-0">{connections.length}</span>
      </button>

      <button
        onclick={() => (activeTab = 'pubsub')}
        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-semibold transition-colors {activeTab === 'pubsub' ? 'bg-redora-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Radio class="h-4 w-4 shrink-0" />
        <span class="truncate text-left flex-1">Pub/Sub Inspector</span>
      </button>

      <button
        onclick={() => (activeTab = 'console')}
        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-semibold transition-colors {activeTab === 'console' ? 'bg-redora-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Terminal class="h-4 w-4 shrink-0" />
        <span class="truncate text-left flex-1">CLI Console</span>
      </button>

      <button
        onclick={() => (activeTab = 'settings')}
        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-semibold transition-colors {activeTab === 'settings' ? 'bg-redora-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <Settings class="h-4 w-4 shrink-0" />
        <span class="truncate text-left flex-1">Settings</span>
      </button>
    </nav>

    <!-- Sidebar Footer -->
    <div class="p-4 border-t border-slate-800 dark:border-dark-border text-xs text-slate-400 flex items-center justify-between shrink-0">
      <div class="flex items-center gap-2">
        <Shield class="h-3.5 w-3.5 text-emerald-400 shrink-0" />
        <span>Self-hosted</span>
      </div>
      <span class="font-mono text-[11px] text-slate-400">SQLite + Go</span>
    </div>
  </aside>

  <!-- Main Workspace Area -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
    <!-- Header Bar -->
    <header class="h-16 border-b border-slate-200 dark:border-dark-border bg-white dark:bg-dark-sidebar/50 backdrop-blur px-6 flex items-center justify-between shrink-0 transition-colors">
      <div class="flex items-center gap-3 min-w-0">
        <h2 class="font-semibold text-slate-900 dark:text-white capitalize shrink-0">{activeTab.replace('-', ' ')}</h2>
        <span class="text-slate-400 shrink-0">/</span>
        {#if activeConn}
          <span class="text-xs text-slate-500 dark:text-slate-400 font-mono bg-slate-100 dark:bg-slate-800 px-2.5 py-1 rounded-lg border border-slate-200 dark:border-slate-700 truncate font-semibold">
            {activeConn.name} ({activeConn.host}:{activeConn.port})
          </span>
        {:else}
          <span class="text-xs text-slate-500">No Connection Selected</span>
        {/if}
      </div>

      <!-- Controls & Health Indicator -->
      <div class="flex items-center gap-3 shrink-0">
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
            <span>Online ({latencyMs}ms)</span>
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
    <main class="flex-1 overflow-hidden flex">
      <!-- TAB 1: KEY BROWSER & VALUE CRUD SPLIT VIEW -->
      {#if activeTab === 'key-browser'}
        <div class="flex-1 flex h-full overflow-hidden">
          <KeyBrowser
            {activeConn}
            {selectedKey}
            onSelectKey={(k) => (selectedKey = k)}
          />
          <ValueViewer
            {activeConn}
            keyName={selectedKey}
            onKeyDeleted={() => (selectedKey = null)}
          />
        </div>
      {/if}

      <!-- TAB 2: CONNECTIONS MANAGEMENT -->
      {#if activeTab === 'connections'}
        <div class="flex-1 p-6 overflow-y-auto space-y-6">
          <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-4 shadow-sm">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="font-semibold text-slate-900 dark:text-white text-base">Configured Redis Connections</h3>
                <p class="text-xs text-slate-500 dark:text-slate-400">Manage, test, and inspect your saved Redis database instances.</p>
              </div>
              <button
                onclick={() => { resetForm(); activeTab = 'add-connection'; }}
                class="px-4 py-2 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition flex items-center gap-2"
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
              <div class="p-12 text-center border-2 border-dashed border-slate-200 dark:border-dark-border rounded-xl space-y-3">
                <Radio class="h-10 w-10 text-slate-400 mx-auto" />
                <div class="font-medium text-slate-700 dark:text-slate-300">No Redis connections saved yet</div>
                <button
                  onclick={() => { resetForm(); activeTab = 'add-connection'; }}
                  class="px-4 py-2 rounded-lg bg-redora-600 text-white text-xs font-semibold hover:bg-redora-500 transition inline-flex items-center gap-2"
                >
                  <PlusCircle class="h-4 w-4" />
                  <span>Add Connection Now</span>
                </button>
              </div>
            {:else}
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
                          {#if activeConnId === conn.id}
                            <span class="px-2 py-0.5 rounded text-[10px] bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-bold">Active</span>
                          {/if}
                        </div>
                        <div class="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-2 mt-0.5">
                          <span>DB {conn.db}</span>
                          <span>&bull;</span>
                          <span>User: {conn.username || 'default'}</span>
                        </div>
                      </div>
                    </div>

                    <div class="flex items-center gap-2 self-end md:self-auto">
                      {#if connTestResults[conn.id]}
                        <span class="text-xs px-2.5 py-1 rounded border font-medium {connTestResults[conn.id].success ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20' : 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'}">
                          {connTestResults[conn.id].message}
                        </span>
                      {/if}

                      <button
                        onclick={() => { activeConnId = conn.id; activeTab = 'key-browser'; }}
                        class="px-3 py-1.5 rounded-md bg-redora-600 hover:bg-redora-500 text-white text-xs font-medium transition"
                      >
                        Browse Keys
                      </button>
                      <button
                        onclick={() => handleTestSavedConn(conn)}
                        disabled={testingConnId === conn.id}
                        class="px-3 py-1.5 rounded-md bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border text-xs text-slate-700 dark:text-slate-300 hover:border-slate-400 transition flex items-center gap-1.5 disabled:opacity-50"
                      >
                        <Play class="h-3.5 w-3.5 text-emerald-500 {testingConnId === conn.id ? 'animate-spin' : ''}" />
                        <span>Test</span>
                      </button>
                      <button
                        onclick={() => startEdit(conn)}
                        class="p-1.5 rounded-md bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-400 hover:text-slate-900"
                        title="Edit Connection"
                      >
                        <Edit3 class="h-3.5 w-3.5" />
                      </button>
                      <button
                        onclick={() => handleDeleteConn(conn)}
                        class="p-1.5 rounded-md bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-400 hover:text-red-600"
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
        </div>
      {/if}

      <!-- TAB 3: ADD / EDIT CONNECTION FORM -->
      {#if activeTab === 'add-connection'}
        <div class="flex-1 p-6 overflow-y-auto">
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
                  class="px-5 py-2 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition flex items-center gap-2 disabled:opacity-50"
                >
                  <Save class="h-3.5 w-3.5" />
                  <span>{formSubmitting ? 'Saving...' : editingId ? 'Update Connection' : 'Save Connection'}</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      {/if}

      <!-- TAB 4: PUB/SUB INSPECTOR -->
      {#if activeTab === 'pubsub'}
        <PubSubInspector activeConn={activeConn} />
      {/if}

      <!-- TAB 5: CLI CONSOLE -->
      {#if activeTab === 'console'}
        <div class="flex-1 p-6 overflow-y-auto">
          <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-4 flex flex-col h-[650px] shadow-sm">
            <div class="flex items-center justify-between gap-3 shrink-0">
              <div>
                <h3 class="font-semibold text-slate-900 dark:text-white text-base flex items-center gap-2">
                  <Terminal class="h-5 w-5 text-redora-500" />
                  <span>Redis Command Console</span>
                </h3>
                <p class="text-xs text-slate-500 dark:text-slate-400">Direct query execution against active connection <code class="font-mono text-redora-500 font-bold">{activeConn?.name || 'Redis'}</code>.</p>
              </div>
            </div>

            <!-- Console Terminal Window -->
            <div class="flex-1 bg-slate-900 border border-slate-800 rounded-lg p-4 font-mono text-xs overflow-y-auto space-y-3">
              {#if commandLogs.length === 0}
                <div class="p-8 text-center text-slate-500 space-y-1">
                  <div>Type any Redis command below to execute against <code class="text-redora-400">{activeConn?.name || 'Redis'}</code>.</div>
                  <div class="text-[11px] text-slate-600">Examples: <code class="text-slate-400">PING</code>, <code class="text-slate-400">GET user:100</code>, <code class="text-slate-400">TTL session:xyz</code>, <code class="text-slate-400">INFO server</code></div>
                </div>
              {:else}
                {#each commandLogs as item}
                  <div class="space-y-1">
                    <div class="flex items-center justify-between text-slate-400">
                      <div class="flex items-center gap-2">
                        <span class="text-redora-400 font-bold">&gt;</span>
                        <span class="text-white font-semibold">{item.command}</span>
                      </div>
                      <span class="text-[10px] text-slate-500">{item.time}</span>
                    </div>
                    <div class="pl-4 whitespace-pre-wrap rounded p-2 {item.status === 'warning' ? 'bg-amber-500/10 text-amber-300 border border-amber-500/20' : item.status === 'error' ? 'bg-red-500/10 text-red-300 border border-red-500/20' : 'bg-slate-950/80 text-emerald-300'}">
                      {item.result}
                    </div>
                  </div>
                {/each}
              {/if}
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
                disabled={isExecutingCmd}
                class="px-4 py-2.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition flex items-center gap-2 disabled:opacity-50"
              >
                <Send class="h-3.5 w-3.5" />
                <span>{isExecutingCmd ? 'Executing...' : 'Execute'}</span>
              </button>
            </form>
          </div>
        </div>
      {/if}

      <!-- TAB 6: SETTINGS -->
      {#if activeTab === 'settings'}
        <div class="flex-1 p-6 overflow-y-auto">
          <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 space-y-6 max-w-3xl shadow-sm">
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white text-base">Application Settings</h3>
              <p class="text-xs text-slate-500 dark:text-slate-400">Configure visual themes, SCAN bounds, and security parameters.</p>
            </div>

            <div class="space-y-4">
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
            </div>
          </div>
        </div>
      {/if}
    </main>
  </div>
</div>

<!-- Modal Confirmation Dangerous Command -->
{#if showDangerousModal}
  <div class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-white dark:bg-dark-card border border-red-500/30 rounded-xl p-6 w-full max-w-md shadow-2xl space-y-4">
      <div class="flex items-center gap-3 text-red-500 border-b border-slate-200 dark:border-dark-border pb-3">
        <AlertTriangle class="h-6 w-6 shrink-0 animate-pulse" />
        <div>
          <h3 class="font-bold text-slate-900 dark:text-white text-sm">Dangerous Command Warning</h3>
          <p class="text-[11px] text-slate-500">Destructive operation requires explicit confirmation</p>
        </div>
      </div>

      <div class="space-y-2 text-xs text-slate-700 dark:text-slate-300">
        <p>You are about to execute a potentially destructive command on <strong class="font-mono text-slate-900 dark:text-white">{activeConn?.name || 'Redis'}</strong>:</p>
        <div class="p-3 rounded-lg bg-slate-900 text-amber-300 font-mono text-xs break-all">
          {pendingDangerousCmd}
        </div>
        <p class="text-[11px] text-slate-500">This action cannot be undone. Are you sure you want to proceed?</p>
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button
          onclick={() => (showDangerousModal = false)}
          class="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-xs font-medium hover:bg-slate-200 transition"
        >
          Cancel
        </button>
        <button
          onclick={() => handleExecuteCommand(pendingDangerousCmd, true)}
          class="px-4 py-2 rounded-lg bg-red-600 hover:bg-red-500 text-white text-xs font-semibold shadow-md transition"
        >
          Confirm & Execute
        </button>
      </div>
    </div>
  </div>
{/if}
