<script lang="ts">
  import { onMount } from 'svelte';
  import { apiClient, type KeyItem, type RedisConnection, type CreateKeyInput } from '../api/client';
  import {
    Search,
    Filter,
    Plus,
    RefreshCw,
    Key,
    Clock,
    Tag,
    ChevronRight,
    PlusCircle,
    Database,
    X,
    Check
  } from 'lucide-svelte';

  interface Props {
    activeConn: RedisConnection | null;
    selectedKey: string | null;
    onSelectKey: (key: string | null) => void;
  }

  let { activeConn, selectedKey, onSelectKey }: Props = $props();

  let keys = $state<KeyItem[]>([]);
  let nextCursor = $state<number>(0);
  let loading = $state<boolean>(false);
  let searchPattern = $state<string>('*');
  let selectedType = $state<string>(''); // empty = all
  let searchTimeout = $state<any>(null);

  // New Key Modal State
  let showAddModal = $state<boolean>(false);
  let newKeyName = $state<string>('');
  let newKeyType = $state<string>('string');
  let newKeyValue = $state<string>('');
  let newKeyField = $state<string>('');
  let newKeyScore = $state<number>(0);
  let newKeyTTL = $state<number>(-1);
  let isSavingKey = $state<boolean>(false);
  let modalError = $state<string | null>(null);

  const typeOptions = [
    { label: 'All Types', value: '', color: 'bg-slate-700 text-slate-200' },
    { label: 'String', value: 'string', color: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30' },
    { label: 'Hash', value: 'hash', color: 'bg-blue-500/20 text-blue-400 border-blue-500/30' },
    { label: 'List', value: 'list', color: 'bg-amber-500/20 text-amber-400 border-amber-500/30' },
    { label: 'Set', value: 'set', color: 'bg-purple-500/20 text-purple-400 border-purple-500/30' },
    { label: 'ZSet', value: 'zset', color: 'bg-pink-500/20 text-pink-400 border-pink-500/30' },
    { label: 'Stream', value: 'stream', color: 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30' },
  ];

  async function loadKeys(cursor = 0, append = false) {
    if (!activeConn) {
      keys = [];
      nextCursor = 0;
      return;
    }
    loading = true;
    try {
      const res = await apiClient.scanKeys(activeConn.id, searchPattern || '*', selectedType, cursor, 100);
      if (append) {
        keys = [...keys, ...res.keys];
      } else {
        keys = res.keys;
      }
      nextCursor = res.nextCursor;
    } catch (e: any) {
      console.error('Scan error:', e);
      keys = [];
      nextCursor = 0;
    } finally {
      loading = false;
    }
  }

  function handleSearchInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    searchPattern = val;
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      loadKeys(0, false);
    }, 300);
  }

  function selectTypeFilter(typeVal: string) {
    selectedType = typeVal;
    loadKeys(0, false);
  }

  function formatTTL(ttl: number): string {
    if (ttl === -1) return 'No Expire';
    if (ttl === -2) return 'Expired';
    if (ttl < 60) return `${ttl}s`;
    if (ttl < 3600) return `${Math.floor(ttl / 60)}m`;
    if (ttl < 86400) return `${Math.floor(ttl / 3600)}h ${Math.floor((ttl % 3600) / 60)}m`;
    return `${Math.floor(ttl / 86400)}d`;
  }

  function getTypeBadgeColor(t: string): string {
    switch (t.toLowerCase()) {
      case 'string': return 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20';
      case 'hash': return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20';
      case 'list': return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20';
      case 'set': return 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20';
      case 'zset': return 'bg-pink-500/10 text-pink-600 dark:text-pink-400 border-pink-500/20';
      case 'stream': return 'bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border-cyan-500/20';
      default: return 'bg-slate-500/10 text-slate-400 border-slate-500/20';
    }
  }

  async function handleCreateKeySubmit() {
    if (!activeConn || !newKeyName.trim()) {
      modalError = 'Key name is required';
      return;
    }

    isSavingKey = true;
    modalError = null;

    const payload: CreateKeyInput = {
      key: newKeyName.trim(),
      type: newKeyType,
      value: newKeyValue,
      field: newKeyField,
      score: newKeyScore,
      ttl: newKeyTTL > 0 ? newKeyTTL : undefined,
    };

    try {
      await apiClient.createKey(activeConn.id, payload);
      showAddModal = false;
      newKeyName = '';
      newKeyValue = '';
      newKeyField = '';
      loadKeys(0, false);
      onSelectKey(payload.key);
    } catch (e: any) {
      modalError = e?.message || 'Failed to create key';
    } finally {
      isSavingKey = false;
    }
  }

  // Reactive Effect: Reset keys and fetch whenever activeConn ID changes
  $effect(() => {
    const connId = activeConn?.id;
    if (connId) {
      keys = [];
      nextCursor = 0;
      loadKeys(0, false);
    } else {
      keys = [];
      nextCursor = 0;
    }
  });
</script>

<div class="flex flex-col h-full bg-white dark:bg-dark-sidebar border-r border-slate-200 dark:border-dark-border text-slate-800 dark:text-slate-200 w-80 shrink-0">
  <!-- Search & Action Header -->
  <div class="p-4 border-b border-slate-200 dark:border-dark-border space-y-3">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Key class="h-4 w-4 text-redora-500 shrink-0" />
        <span class="font-bold text-sm text-slate-900 dark:text-white">Key Browser</span>
      </div>
      <div class="flex items-center gap-1">
        <button
          onclick={() => loadKeys(0, false)}
          disabled={loading}
          class="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 transition"
          title="Refresh Keys"
        >
          <RefreshCw class="h-3.5 w-3.5 {loading ? 'animate-spin text-redora-500' : ''}" />
        </button>
        <button
          onclick={() => (showAddModal = true)}
          class="px-2.5 py-1 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-medium flex items-center gap-1 shadow-sm transition"
        >
          <Plus class="h-3.5 w-3.5 shrink-0" />
          <span>New Key</span>
        </button>
      </div>
    </div>

    <!-- Search Box -->
    <div class="relative">
      <input
        type="text"
        value={searchPattern}
        oninput={handleSearchInput}
        placeholder="SCAN pattern (e.g. user:*)..."
        class="w-full pl-8 pr-3 py-1.5 rounded-lg bg-slate-100 dark:bg-dark-card border border-slate-200 dark:border-dark-border text-xs text-slate-900 dark:text-white focus:outline-none focus:border-redora-500 font-mono"
      />
      <Search class="h-3.5 w-3.5 text-slate-400 absolute left-2.5 top-2.5" />
    </div>

    <!-- Type Filters -->
    <div class="flex items-center gap-1 overflow-x-auto pb-1 no-scrollbar">
      {#each typeOptions as option}
        <button
          onclick={() => selectTypeFilter(option.value)}
          class="px-2 py-0.5 rounded text-[10px] font-medium border shrink-0 transition {selectedType === option.value ? 'bg-redora-600 text-white border-redora-500 shadow-sm' : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700 hover:border-slate-400'}"
        >
          {option.label}
        </button>
      {/each}
    </div>
  </div>

  <!-- Key List -->
  <div class="flex-1 overflow-y-auto divide-y divide-slate-100 dark:divide-dark-border">
    {#if loading && keys.length === 0}
      <div class="p-6 text-center text-xs text-slate-400 space-y-2">
        <RefreshCw class="h-5 w-5 animate-spin mx-auto text-redora-500" />
        <span>Scanning Redis keys...</span>
      </div>
    {:else if keys.length === 0}
      <div class="p-8 text-center text-xs text-slate-400 space-y-2">
        <Database class="h-8 w-8 mx-auto text-slate-300 dark:text-slate-600" />
        <div>No keys match criteria</div>
        <div class="text-[10px] text-slate-500">Pattern: <code class="font-mono">{searchPattern}</code></div>
      </div>
    {:else}
      {#each keys as item}
        <button
          onclick={() => onSelectKey(item.key)}
          class="w-full text-left p-3 hover:bg-slate-50 dark:hover:bg-dark-hover transition flex items-center justify-between group {selectedKey === item.key ? 'bg-redora-500/10 border-l-2 border-redora-500' : ''}"
        >
          <div class="min-w-0 flex-1 pr-2">
            <div class="font-mono text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">
              {item.key}
            </div>
            <div class="flex items-center gap-2 mt-1">
              <span class="px-1.5 py-0.5 rounded text-[9px] uppercase font-bold border {getTypeBadgeColor(item.type)}">
                {item.type}
              </span>
              <span class="text-[10px] text-slate-500 dark:text-slate-400 flex items-center gap-1 font-mono">
                <Clock class="h-3 w-3" />
                <span>{formatTTL(item.ttl)}</span>
              </span>
            </div>
          </div>
          <ChevronRight class="h-4 w-4 text-slate-400 opacity-0 group-hover:opacity-100 transition shrink-0" />
        </button>
      {/each}

      <!-- Next Cursor Load More -->
      {#if nextCursor > 0}
        <div class="p-3">
          <button
            onclick={() => loadKeys(nextCursor, true)}
            disabled={loading}
            class="w-full py-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 text-slate-700 dark:text-slate-300 text-xs font-medium transition flex items-center justify-center gap-2"
          >
            <span>Load More Keys (Cursor {nextCursor})</span>
          </button>
        </div>
      {/if}
    {/if}
  </div>
</div>

<!-- Modal Create Key -->
{#if showAddModal}
  <div class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 w-full max-w-md shadow-2xl space-y-4">
      <div class="flex items-center justify-between border-b border-slate-200 dark:border-dark-border pb-3">
        <h3 class="font-bold text-slate-900 dark:text-white text-sm">Create New Redis Key</h3>
        <button onclick={() => (showAddModal = false)} class="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200">
          <X class="h-4 w-4" />
        </button>
      </div>

      {#if modalError}
        <div class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 text-xs">
          {modalError}
        </div>
      {/if}

      <div class="space-y-3">
        <div>
          <label for="new-key-name" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Key Name *</label>
          <input
            id="new-key-name"
            type="text"
            bind:value={newKeyName}
            placeholder="e.g. user:100 or session:xyz"
            class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="new-key-type" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Data Type</label>
            <select
              id="new-key-type"
              bind:value={newKeyType}
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white focus:outline-none focus:border-redora-500"
            >
              <option value="string">String</option>
              <option value="hash">Hash</option>
              <option value="list">List</option>
              <option value="set">Set</option>
              <option value="zset">Sorted Set (ZSet)</option>
            </select>
          </div>

          <div>
            <label for="new-key-ttl" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">TTL (Seconds)</label>
            <input
              id="new-key-ttl"
              type="number"
              bind:value={newKeyTTL}
              placeholder="-1 for permanent"
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
            />
          </div>
        </div>

        {#if newKeyType === 'hash'}
          <div>
            <label for="new-key-field" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Hash Field Name</label>
            <input
              id="new-key-field"
              type="text"
              bind:value={newKeyField}
              placeholder="e.g. email or username"
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
            />
          </div>
        {/if}

        {#if newKeyType === 'zset'}
          <div>
            <label for="new-key-score" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Score</label>
            <input
              id="new-key-score"
              type="number"
              bind:value={newKeyScore}
              step="any"
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
            />
          </div>
        {/if}

        <div>
          <label for="new-key-val" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Value</label>
          <textarea
            id="new-key-val"
            bind:value={newKeyValue}
            rows="3"
            placeholder="Key value string / JSON content..."
            class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
          ></textarea>
        </div>
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button
          onclick={() => (showAddModal = false)}
          class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-xs font-medium hover:bg-slate-200 transition"
        >
          Cancel
        </button>
        <button
          onclick={handleCreateKeySubmit}
          disabled={isSavingKey}
          class="px-4 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition disabled:opacity-50"
        >
          {isSavingKey ? 'Saving...' : 'Create Key'}
        </button>
      </div>
    </div>
  </div>
{/if}
