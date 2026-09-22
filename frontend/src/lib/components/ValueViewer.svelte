<script lang="ts">
  import { onMount } from 'svelte';
  import { apiClient, type KeyDetailResponse, type RedisConnection } from '../api/client';
  import {
    Clock,
    Tag,
    Trash2,
    Edit3,
    Save,
    RefreshCw,
    Code,
    Plus,
    X,
    Check,
    AlertTriangle,
    Layers,
    ListFilter,
    FileText
  } from 'lucide-svelte';

  interface Props {
    activeConn: RedisConnection | null;
    keyName: string | null;
    onKeyDeleted: () => void;
  }

  let { activeConn, keyName, onKeyDeleted }: Props = $props();

  let detail = $state<KeyDetailResponse | null>(null);
  let loading = $state<boolean>(false);
  let error = $state<string | null>(null);

  // String Editor State
  let stringValue = $state<string>('');
  let isJsonFormatted = $state<boolean>(false);
  let isSavingValue = $state<boolean>(false);
  let updateSuccess = $state<boolean>(false);

  // Hash/Set/ZSet Add Field Modal State
  let showAddFieldModal = $state<boolean>(false);
  let addFieldName = $state<string>('');
  let addFieldValue = $state<string>('');
  let addFieldScore = $state<number>(0);

  // TTL Edit Modal State
  let showTTLModal = $state<boolean>(false);
  let ttlSecondsInput = $state<number>(-1);

  // Rename Key Modal State
  let showRenameModal = $state<boolean>(false);
  let newKeyInput = $state<string>('');

  async function fetchKeyDetail() {
    if (!activeConn || !keyName) return;
    loading = true;
    error = null;
    try {
      detail = await apiClient.getKeyDetail(activeConn.id, keyName);
      if (detail.type === 'string') {
        stringValue = String(detail.value);
        checkAndFormatJSON();
      }
    } catch (e: any) {
      error = e?.message || 'Failed to load key details';
    } finally {
      loading = false;
    }
  }

  function checkAndFormatJSON() {
    try {
      const parsed = JSON.parse(stringValue);
      stringValue = JSON.stringify(parsed, null, 2);
      isJsonFormatted = true;
    } catch {
      isJsonFormatted = false;
    }
  }

  async function handleSaveStringValue() {
    if (!activeConn || !keyName) return;
    isSavingValue = true;
    try {
      await apiClient.updateKey(activeConn.id, keyName, {
        key: keyName,
        type: 'string',
        value: stringValue,
      });
      updateSuccess = true;
      setTimeout(() => (updateSuccess = false), 2500);
      fetchKeyDetail();
    } catch (e: any) {
      alert(`Update failed: ${e?.message}`);
    } finally {
      isSavingValue = false;
    }
  }

  async function handleAddFieldSubmit() {
    if (!activeConn || !keyName || !detail) return;
    try {
      await apiClient.updateKey(activeConn.id, keyName, {
        key: keyName,
        type: detail.type,
        field: addFieldName,
        value: addFieldValue,
        score: addFieldScore,
      });
      showAddFieldModal = false;
      addFieldName = '';
      addFieldValue = '';
      fetchKeyDetail();
    } catch (e: any) {
      alert(`Failed to add item: ${e?.message}`);
    }
  }

  async function handleSaveTTL() {
    if (!activeConn || !keyName) return;
    try {
      await apiClient.setTTL(activeConn.id, keyName, ttlSecondsInput);
      showTTLModal = false;
      fetchKeyDetail();
    } catch (e: any) {
      alert(`Failed to set TTL: ${e?.message}`);
    }
  }

  async function handleRenameSubmit() {
    if (!activeConn || !keyName || !newKeyInput.trim()) return;
    try {
      await apiClient.renameKey(activeConn.id, keyName, newKeyInput.trim());
      showRenameModal = false;
      onKeyDeleted();
    } catch (e: any) {
      alert(`Failed to rename key: ${e?.message}`);
    }
  }

  async function handleDeleteKey() {
    if (!activeConn || !keyName) return;
    if (!confirm(`Are you sure you want to delete key "${keyName}"?`)) return;
    try {
      await apiClient.deleteKey(activeConn.id, keyName);
      onKeyDeleted();
    } catch (e: any) {
      alert(`Delete failed: ${e?.message}`);
    }
  }

  function formatHumanTTL(ttl: number): string {
    if (ttl === -1) return 'No Expiration (Persistent)';
    if (ttl === -2) return 'Key Expired / Not Found';
    if (ttl < 60) return `${ttl} Seconds`;
    if (ttl < 3600) return `${Math.floor(ttl / 60)}m ${ttl % 60}s`;
    if (ttl < 86400) return `${Math.floor(ttl / 3600)}h ${Math.floor((ttl % 3600) / 60)}m`;
    return `${Math.floor(ttl / 86400)} Days`;
  }

  $effect(() => {
    if (keyName && activeConn) {
      fetchKeyDetail();
    }
  });
</script>

<div class="flex-1 flex flex-col h-full bg-slate-50 dark:bg-dark-bg overflow-hidden">
  {#if !keyName}
    <div class="flex-1 flex items-center justify-center p-8 text-center text-slate-400">
      <div class="space-y-3 max-w-sm">
        <FileText class="h-12 w-12 mx-auto text-slate-300 dark:text-slate-600" />
        <div class="font-medium text-slate-700 dark:text-slate-300">No Key Selected</div>
        <p class="text-xs text-slate-500">Select a Redis key from the SCAN browser list to inspect and edit its value.</p>
      </div>
    </div>
  {:else if loading}
    <div class="flex-1 flex items-center justify-center p-8 text-center text-slate-400">
      <div class="space-y-2">
        <RefreshCw class="h-6 w-6 animate-spin mx-auto text-redora-500" />
        <span class="text-xs">Loading key value...</span>
      </div>
    </div>
  {:else if error}
    <div class="flex-1 p-6">
      <div class="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 text-xs">
        {error}
      </div>
    </div>
  {:else if detail}
    <!-- Key Header Bar -->
    <div class="p-5 border-b border-slate-200 dark:border-dark-border bg-white dark:bg-dark-card space-y-3 shrink-0 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
        <div class="space-y-1 min-w-0">
          <div class="flex items-center gap-2">
            <span class="px-2 py-0.5 rounded text-[10px] uppercase font-bold border bg-redora-500/10 text-redora-600 dark:text-redora-400 border-redora-500/20">
              {detail.type}
            </span>
            <h2 class="font-mono text-base font-bold text-slate-900 dark:text-white truncate" title={detail.key}>
              {detail.key}
            </h2>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="flex items-center gap-2 shrink-0">
          <button
            onclick={() => { newKeyInput = detail!.key; showRenameModal = true; }}
            class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-xs text-slate-700 dark:text-slate-300 font-medium transition flex items-center gap-1.5"
          >
            <Edit3 class="h-3.5 w-3.5" />
            <span>Rename</span>
          </button>
          <button
            onclick={handleDeleteKey}
            class="px-3 py-1.5 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-xs text-red-600 dark:text-red-400 border border-red-500/20 font-medium transition flex items-center gap-1.5"
          >
            <Trash2 class="h-3.5 w-3.5" />
            <span>Delete</span>
          </button>
        </div>
      </div>

      <!-- Metadata & TTL Info Bar -->
      <div class="flex flex-wrap items-center gap-4 text-xs text-slate-600 dark:text-slate-400 pt-2 border-t border-slate-100 dark:border-slate-800">
        <div class="flex items-center gap-1.5">
          <Clock class="h-3.5 w-3.5 text-amber-500" />
          <span>TTL: <strong class="font-mono text-slate-900 dark:text-slate-200">{formatHumanTTL(detail.ttl)}</strong></span>
          <button
            onclick={() => { ttlSecondsInput = detail!.ttl; showTTLModal = true; }}
            class="ml-1 text-[11px] text-redora-500 hover:underline font-medium"
          >
            Edit TTL
          </button>
        </div>
      </div>
    </div>

    <!-- Type Specific Value Viewers -->
    <div class="flex-1 p-6 overflow-y-auto">
      <!-- 1. STRING VIEWER -->
      {#if detail.type === 'string'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 space-y-4 shadow-sm">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">String Value Editor</h3>
            <div class="flex items-center gap-2">
              <button
                onclick={checkAndFormatJSON}
                class="px-3 py-1 rounded-md bg-slate-100 dark:bg-slate-800 text-xs text-slate-700 dark:text-slate-300 hover:text-slate-900 transition flex items-center gap-1.5"
              >
                <Code class="h-3.5 w-3.5 text-cyan-500" />
                <span>Format JSON</span>
              </button>
            </div>
          </div>

          <textarea
            bind:value={stringValue}
            rows="12"
            class="w-full p-4 rounded-lg bg-slate-900 border border-slate-700 text-xs font-mono text-emerald-300 focus:outline-none focus:border-redora-500 leading-relaxed resize-y"
          ></textarea>

          <div class="flex items-center justify-between pt-2">
            {#if updateSuccess}
              <span class="text-xs text-emerald-500 font-medium flex items-center gap-1">
                <Check class="h-4 w-4" />
                <span>Value updated successfully!</span>
              </span>
            {:else}
              <span></span>
            {/if}

            <button
              onclick={handleSaveStringValue}
              disabled={isSavingValue}
              class="px-4 py-2 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition flex items-center gap-2 disabled:opacity-50"
            >
              <Save class="h-3.5 w-3.5" />
              <span>{isSavingValue ? 'Saving...' : 'Save Changes'}</span>
            </button>
          </div>
        </div>

      <!-- 2. HASH TABLE VIEWER -->
      {:else if detail.type === 'hash'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 space-y-4 shadow-sm">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Hash Fields & Values</h3>
            <button
              onclick={() => (showAddFieldModal = true)}
              class="px-3 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-sm transition flex items-center gap-1.5"
            >
              <Plus class="h-3.5 w-3.5" />
              <span>Add Hash Field</span>
            </button>
          </div>

          <div class="border border-slate-200 dark:border-dark-border rounded-lg overflow-hidden">
            <table class="w-full text-left border-collapse text-xs">
              <thead>
                <tr class="bg-slate-100 dark:bg-dark-sidebar border-b border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-400 font-semibold">
                  <th class="p-3 w-1/3">Field</th>
                  <th class="p-3">Value</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-dark-border font-mono">
                {#each Object.entries(detail.value || {}) as [f, v]}
                  <tr class="hover:bg-slate-50 dark:hover:bg-dark-hover">
                    <td class="p-3 font-bold text-blue-600 dark:text-blue-400">{f}</td>
                    <td class="p-3 text-slate-800 dark:text-slate-200 break-all">{v}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>

      <!-- 3. LIST VIEWER -->
      {:else if detail.type === 'list'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 space-y-4 shadow-sm">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Indexed List Items</h3>
            <button
              onclick={() => (showAddFieldModal = true)}
              class="px-3 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-sm transition flex items-center gap-1.5"
            >
              <Plus class="h-3.5 w-3.5" />
              <span>Push Item (RPUSH)</span>
            </button>
          </div>

          <div class="space-y-2">
            {#each (detail.value || []) as item, idx}
              <div class="p-3 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-200 dark:border-dark-border flex items-center gap-3 font-mono text-xs">
                <span class="px-2 py-0.5 rounded bg-slate-200 dark:bg-slate-800 text-slate-600 dark:text-slate-400 text-[10px] font-bold">[{idx}]</span>
                <span class="text-slate-800 dark:text-slate-200 break-all">{item}</span>
              </div>
            {/each}
          </div>
        </div>

      <!-- 4. SET VIEWER -->
      {:else if detail.type === 'set'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 space-y-4 shadow-sm">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Set Members</h3>
            <button
              onclick={() => (showAddFieldModal = true)}
              class="px-3 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-sm transition flex items-center gap-1.5"
            >
              <Plus class="h-3.5 w-3.5" />
              <span>Add Member (SADD)</span>
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
            {#each (detail.value || []) as member}
              <div class="p-3 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-200 dark:border-dark-border font-mono text-xs text-purple-600 dark:text-purple-400 break-all">
                {member}
              </div>
            {/each}
          </div>
        </div>

      <!-- 5. SORTED SET (ZSET) VIEWER -->
      {:else if detail.type === 'zset'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 space-y-4 shadow-sm">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Sorted Set Members & Scores</h3>
            <button
              onclick={() => (showAddFieldModal = true)}
              class="px-3 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-sm transition flex items-center gap-1.5"
            >
              <Plus class="h-3.5 w-3.5" />
              <span>Add Member (ZADD)</span>
            </button>
          </div>

          <div class="border border-slate-200 dark:border-dark-border rounded-lg overflow-hidden">
            <table class="w-full text-left border-collapse text-xs">
              <thead>
                <tr class="bg-slate-100 dark:bg-dark-sidebar border-b border-slate-200 dark:border-dark-border text-slate-600 dark:text-slate-400 font-semibold">
                  <th class="p-3 w-1/4">Score</th>
                  <th class="p-3">Member</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-dark-border font-mono">
                {#each (detail.value || []) as zItem}
                  <tr class="hover:bg-slate-50 dark:hover:bg-dark-hover">
                    <td class="p-3 font-bold text-pink-600 dark:text-pink-400">{zItem.score}</td>
                    <td class="p-3 text-slate-800 dark:text-slate-200 break-all">{zItem.member}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>

      <!-- 6. STREAM VIEWER -->
      {:else if detail.type === 'stream'}
        <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 space-y-4 shadow-sm">
          <h3 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Stream Entries (XREAD / XRANGE)</h3>

          <div class="space-y-3">
            {#each (detail.value || []) as entry}
              <div class="p-4 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-200 dark:border-dark-border space-y-2 font-mono text-xs">
                <div class="font-bold text-cyan-600 dark:text-cyan-400">ID: {entry.id}</div>
                <div class="pl-3 border-l-2 border-slate-300 dark:border-slate-700 space-y-1">
                  {#each Object.entries(entry.values || {}) as [k, v]}
                    <div><span class="text-slate-500">{k}:</span> <span class="text-slate-800 dark:text-slate-200">{v}</span></div>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Modal Add Field/Member -->
{#if showAddFieldModal && detail}
  <div class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 w-full max-w-md shadow-2xl space-y-4">
      <div class="flex items-center justify-between border-b border-slate-200 dark:border-dark-border pb-3">
        <h3 class="font-bold text-slate-900 dark:text-white text-sm">Add Item to {detail.type.toUpperCase()}</h3>
        <button onclick={() => (showAddFieldModal = false)} class="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200">
          <X class="h-4 w-4" />
        </button>
      </div>

      <div class="space-y-3">
        {#if detail.type === 'hash'}
          <div>
            <label for="add-field-name" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Field Name *</label>
            <input
              id="add-field-name"
              type="text"
              bind:value={addFieldName}
              placeholder="Field key"
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono"
            />
          </div>
        {/if}

        {#if detail.type === 'zset'}
          <div>
            <label for="add-field-score" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Score *</label>
            <input
              id="add-field-score"
              type="number"
              bind:value={addFieldScore}
              step="any"
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono"
            />
          </div>
        {/if}

        <div>
          <label for="add-field-value" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Value / Member *</label>
          <input
            id="add-field-value"
            type="text"
            bind:value={addFieldValue}
            placeholder="Content value"
            class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono"
          />
        </div>
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button
          onclick={() => (showAddFieldModal = false)}
          class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-xs font-medium hover:bg-slate-200 transition"
        >
          Cancel
        </button>
        <button
          onclick={handleAddFieldSubmit}
          class="px-4 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition"
        >
          Add Item
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Edit TTL -->
{#if showTTLModal}
  <div class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 w-full max-w-sm shadow-2xl space-y-4">
      <div class="flex items-center justify-between border-b border-slate-200 dark:border-dark-border pb-3">
        <h3 class="font-bold text-slate-900 dark:text-white text-sm">Set Key Expiration (TTL)</h3>
        <button onclick={() => (showTTLModal = false)} class="text-slate-400 hover:text-slate-600">
          <X class="h-4 w-4" />
        </button>
      </div>

      <div>
        <label for="ttl-input" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">TTL in Seconds (-1 to persist)</label>
        <input
          id="ttl-input"
          type="number"
          bind:value={ttlSecondsInput}
          placeholder="e.g. 3600 for 1 hour"
          class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono"
        />
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button onclick={() => (showTTLModal = false)} class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-xs font-medium text-slate-700 dark:text-slate-300">
          Cancel
        </button>
        <button onclick={handleSaveTTL} class="px-4 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md">
          Save TTL
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal Rename Key -->
{#if showRenameModal}
  <div class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 w-full max-w-sm shadow-2xl space-y-4">
      <div class="flex items-center justify-between border-b border-slate-200 dark:border-dark-border pb-3">
        <h3 class="font-bold text-slate-900 dark:text-white text-sm">Rename Key</h3>
        <button onclick={() => (showRenameModal = false)} class="text-slate-400 hover:text-slate-600">
          <X class="h-4 w-4" />
        </button>
      </div>

      <div>
        <label for="rename-input" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">New Key Name *</label>
        <input
          id="rename-input"
          type="text"
          bind:value={newKeyInput}
          class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono"
        />
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button onclick={() => (showRenameModal = false)} class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-xs font-medium text-slate-700 dark:text-slate-300">
          Cancel
        </button>
        <button onclick={handleRenameSubmit} class="px-4 py-1.5 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md">
          Rename
        </button>
      </div>
    </div>
  </div>
{/if}
