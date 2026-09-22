<script lang="ts">
  import { onMount } from 'svelte';
  import { apiClient, type RedisConnection } from '../api/client';
  import {
    Radio,
    Send,
    RefreshCw,
    MessageSquare,
    Search,
    CheckCircle2,
    XCircle,
    Volume2
  } from 'lucide-svelte';

  interface Props {
    activeConn: RedisConnection | null;
  }

  let { activeConn }: Props = $props();

  let channels = $state<string[]>([]);
  let loadingChannels = $state<boolean>(false);
  let channelPattern = $state<string>('*');

  // Publish Form State
  let publishChannel = $state<string>('notifications');
  let publishMessage = $state<string>('{"event": "user_logged_in", "user_id": 99}');
  let isPublishing = $state<boolean>(false);
  let publishStatus = $state<{ success: boolean; text: string } | null>(null);

  // Live Feed Log
  let pubSubFeed = $state<Array<{ channel: string; message: string; time: string; subscribers: number }>>([]);

  async function loadChannels() {
    if (!activeConn) return;
    loadingChannels = true;
    try {
      const res = await apiClient.getPubSubChannels(activeConn.id, channelPattern || '*');
      channels = res.channels;
    } catch (e: any) {
      console.error('PubSub channel fetch error:', e);
    } finally {
      loadingChannels = false;
    }
  }

  async function handlePublishSubmit() {
    if (!activeConn || !publishChannel.trim() || !publishMessage.trim()) {
      publishStatus = { success: false, text: 'Channel name and message are required' };
      return;
    }

    isPublishing = true;
    publishStatus = null;

    try {
      const res = await apiClient.publishMessage(activeConn.id, publishChannel.trim(), publishMessage.trim());
      publishStatus = {
        success: true,
        text: `Message published to "${publishChannel}" (${res.subscribers} active subscriber(s))`
      };

      pubSubFeed.unshift({
        channel: publishChannel.trim(),
        message: publishMessage.trim(),
        time: new Date().toLocaleTimeString(),
        subscribers: res.subscribers,
      });

      loadChannels();
    } catch (e: any) {
      publishStatus = { success: false, text: e?.message || 'Publishing failed' };
    } finally {
      isPublishing = false;
    }
  }

  $effect(() => {
    if (activeConn) {
      loadChannels();
    }
  });
</script>

<div class="flex-1 p-6 space-y-6 bg-slate-50 dark:bg-dark-bg overflow-y-auto">
  <!-- Header -->
  <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-6 shadow-sm space-y-2">
    <div class="flex items-center justify-between">
      <div>
        <h3 class="font-bold text-slate-900 dark:text-white text-base flex items-center gap-2">
          <Radio class="h-5 w-5 text-redora-500" />
          <span>Redis Pub/Sub Inspector</span>
        </h3>
        <p class="text-xs text-slate-500 dark:text-slate-400">Inspect active channels (`PUBSUB CHANNELS`) and publish real-time event messages.</p>
      </div>
      <button
        onclick={loadChannels}
        disabled={loadingChannels}
        class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 text-xs text-slate-700 dark:text-slate-300 font-medium transition flex items-center gap-1.5"
      >
        <RefreshCw class="h-3.5 w-3.5 {loadingChannels ? 'animate-spin text-redora-500' : ''}" />
        <span>Refresh Channels</span>
      </button>
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    <!-- Active Channels Panel -->
    <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 shadow-sm space-y-4">
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-dark-border pb-3">
        <span class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Active Channels</span>
        <span class="px-2 py-0.5 rounded text-[10px] bg-slate-100 dark:bg-slate-800 font-mono font-bold text-slate-700 dark:text-slate-300">{channels.length}</span>
      </div>

      <div class="space-y-2">
        {#if loadingChannels}
          <div class="p-4 text-center text-xs text-slate-400">Loading channels...</div>
        {:else if channels.length === 0}
          <div class="p-6 text-center text-xs text-slate-400 space-y-1">
            <Volume2 class="h-6 w-6 mx-auto text-slate-300 dark:text-slate-600" />
            <div>No active channels detected</div>
          </div>
        {:else}
          {#each channels as ch}
            <button
              onclick={() => (publishChannel = ch)}
              class="w-full text-left p-2.5 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-200 dark:border-dark-border hover:border-redora-500 transition text-xs font-mono text-slate-800 dark:text-slate-200 flex items-center justify-between"
            >
              <span>{ch}</span>
              <span class="text-[10px] text-redora-500 font-sans font-medium">Select &rarr;</span>
            </button>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Publish Message Form & Live Feed -->
    <div class="md:col-span-2 space-y-6">
      <!-- Publish Form -->
      <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 shadow-sm space-y-4">
        <h4 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Publish Message to Channel</h4>

        {#if publishStatus}
          <div class="p-3 rounded-lg text-xs font-medium flex items-center gap-2 {publishStatus.success ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20' : 'bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/20'}">
            {#if publishStatus.success}
              <CheckCircle2 class="h-4 w-4 text-emerald-500 shrink-0" />
            {:else}
              <XCircle class="h-4 w-4 text-red-500 shrink-0" />
            {/if}
            <span>{publishStatus.text}</span>
          </div>
        {/if}

        <form onsubmit={(e) => { e.preventDefault(); handlePublishSubmit(); }} class="space-y-3">
          <div>
            <label for="pub-channel" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Target Channel Name *</label>
            <input
              id="pub-channel"
              type="text"
              bind:value={publishChannel}
              placeholder="e.g. notifications or user_events"
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
            />
          </div>

          <div>
            <label for="pub-message" class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Message Payload *</label>
            <textarea
              id="pub-message"
              bind:value={publishMessage}
              rows="3"
              placeholder="JSON or raw message string..."
              class="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-dark-sidebar border border-slate-300 dark:border-dark-border text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-redora-500"
            ></textarea>
          </div>

          <div class="flex justify-end">
            <button
              type="submit"
              disabled={isPublishing}
              class="px-4 py-2 rounded-lg bg-redora-600 hover:bg-redora-500 text-white text-xs font-semibold shadow-md transition flex items-center gap-2 disabled:opacity-50"
            >
              <Send class="h-3.5 w-3.5" />
              <span>{isPublishing ? 'Publishing...' : 'Publish Message'}</span>
            </button>
          </div>
        </form>
      </div>

      <!-- Published Feed History -->
      <div class="bg-white dark:bg-dark-card border border-slate-200 dark:border-dark-border rounded-xl p-5 shadow-sm space-y-3">
        <h4 class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400 tracking-wider">Message Activity Feed</h4>

        {#if pubSubFeed.length === 0}
          <div class="p-6 text-center text-xs text-slate-400">No published messages in this session yet.</div>
        {:else}
          <div class="space-y-2">
            {#each pubSubFeed as item}
              <div class="p-3 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono space-y-1">
                <div class="flex items-center justify-between text-slate-400">
                  <div class="flex items-center gap-2">
                    <span class="text-redora-400 font-bold">[{item.channel}]</span>
                    <span class="text-slate-500">({item.subscribers} subscriber(s))</span>
                  </div>
                  <span class="text-[10px] text-slate-500">{item.time}</span>
                </div>
                <div class="text-emerald-300 break-all pl-2 border-l-2 border-emerald-500/40">{item.message}</div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
