<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { onWS } from '$lib/ws';
  import { tr } from '$lib/i18n';

  const LEVELS = ['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'] as const;

  interface LogEntry {
    time: string;
    level: string;
    message: string;
  }

  let entries: LogEntry[] = $state([]);
  let activeLevels: Set<string> = $state(new Set(LEVELS));
  let autoScroll = $state(true);
  let container: HTMLDivElement;
  let unsubWS: () => void;

  const levelStyles: Record<string, { text: string; bg: string; dot: string }> = {
    trace: { text: 'text-dim', bg: '', dot: 'bg-dim' },
    debug: { text: 'text-dim', bg: '', dot: 'bg-dim' },
    info: { text: 'text-dim-light', bg: '', dot: 'bg-accent' },
    warn: { text: 'text-amber-400', bg: 'bg-amber-400/5', dot: 'bg-amber-400' },
    error: { text: 'text-red', bg: 'bg-red/5', dot: 'bg-red' },
    fatal: { text: 'text-red', bg: 'bg-red/10', dot: 'bg-red' },
    panic: { text: 'text-red', bg: 'bg-red/10', dot: 'bg-red' },
  };

  function toggleLevel(level: string) {
    const next = new Set(activeLevels);
    if (next.has(level)) next.delete(level);
    else next.add(level);
    activeLevels = next;
  }

  const filtered = $derived(entries.filter(e => activeLevels.has(e.level)));

  async function loadScrollback() {
    try {
      const token = localStorage.getItem('mist_token');
      const resp = await fetch('/api/logs', {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      if (resp.ok) {
        const data = await resp.json();
        entries = data.entries || [];
        scrollBottom();
      }
    } catch { /* ignore */ }
  }

  function scrollBottom() {
    if (autoScroll && container) {
      requestAnimationFrame(() => {
        container.scrollTop = container.scrollHeight;
      });
    }
  }

  function handleScroll() {
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    autoScroll = scrollHeight - scrollTop - clientHeight < 40;
  }

  onMount(() => {
    loadScrollback();

    unsubWS = onWS('log', (msg) => {
      const payload = msg.payload as Record<string, unknown>;
      const entry: LogEntry = {
        time: (payload.time as string) || new Date(msg.time).toISOString(),
        level: (payload.level as string) ?? 'info',
        message: (payload.message as string) ?? JSON.stringify(msg.payload)
      };
      entries = [...entries, entry].slice(-500);
      scrollBottom();
    });
  });

  onDestroy(() => {
    unsubWS?.();
  });
</script>

<div class="card flex flex-col h-full overflow-hidden">
  <div class="flex items-center justify-between mb-4 gap-2 flex-wrap">
    <h3 class="text-xs font-medium text-dim uppercase tracking-wider">{$tr('logs.liveLogs')}</h3>
    <div class="flex items-center gap-1 flex-wrap">
      {#each LEVELS as lvl}
        <button
          onclick={() => toggleLevel(lvl)}
          class="text-[10px] font-medium px-1.5 py-0.5 rounded-md border transition-all duration-150 cursor-pointer
            {activeLevels.has(lvl)
              ? 'border-border-light bg-surface-alt text-dim-light'
              : 'border-transparent text-dim/50'}"
        >
          {lvl.toUpperCase()}
        </button>
      {/each}
      <span class="text-[10px] text-dim ml-1.5 tabular-nums">{filtered.length} {$tr('logs.lines')}</span>
    </div>
  </div>

  <div
    class="flex-1 overflow-y-auto font-mono text-xs leading-relaxed bg-bg rounded-lg border border-border p-3 min-h-[300px] max-h-[600px]"
    bind:this={container}
    onscroll={handleScroll}
  >
    {#if filtered.length === 0}
      <div class="text-dim text-center py-12">{$tr('logs.noEntries')}</div>
    {:else}
      {#each filtered as entry (entry.time + entry.message)}
        <div class="flex gap-2.5 py-px hover:bg-surface/50 rounded-sm -mx-1 px-1 transition-colors duration-75">
          <span class="text-dim/60 shrink-0 whitespace-nowrap select-none">{entry.time.slice(11, 23)}</span>
          <span class="shrink-0 w-10 text-right font-medium {levelStyles[entry.level]?.text ?? 'text-dim-light'}">{entry.level}</span>
          <span class="{levelStyles[entry.level]?.text ?? 'text-dim-light'} break-all">{entry.message}</span>
        </div>
      {/each}
    {/if}
  </div>
</div>
