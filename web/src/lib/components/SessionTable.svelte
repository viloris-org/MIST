<script lang="ts">
  import type { SessionInfo } from '$lib/types';
  import { tr } from '$lib/i18n';

  interface Props {
    sessions: SessionInfo[];
  }

  let { sessions }: Props = $props();

  function fmtDuration(ms: number): string {
    const s = Math.floor(ms / 1000);
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m ${s % 60}s`;
    const h = Math.floor(m / 60);
    return `${h}h ${m % 60}m`;
  }
</script>

<div class="card-elevated overflow-hidden">
  <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('sessions.detail')}</h3>

  {#if sessions.length === 0}
    <p class="text-dim text-sm py-12 text-center">{$tr('sessions.noSessions')}</p>
  {:else}
    <div class="overflow-x-auto -mx-5">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border text-dim text-xs uppercase tracking-wider">
            <th class="text-left py-3 pl-5 pr-4 font-medium">{$tr('sessions.seq')}</th>
            <th class="text-left py-3 pr-4 font-medium">{$tr('sessions.state')}</th>
            <th class="text-left py-3 pr-4 font-medium">{$tr('sessions.age')}</th>
            <th class="text-right py-3 pr-4 font-medium">{$tr('sessions.streams')}</th>
            <th class="text-right py-3 pr-5 font-medium">{$tr('sessions.packets')}</th>
          </tr>
        </thead>
        <tbody>
          {#each sessions as s (s.seq)}
            <tr class="border-b border-border/40 hover:bg-surface-alt/50 transition-colors duration-150">
              <td class="py-3 pl-5 pr-4 font-mono tabular-nums text-text text-xs">{s.seq}</td>
              <td class="py-3 pr-4">
                {#if s.is_closed}
                  <span class="inline-flex items-center gap-1 text-red text-xs">
                    <span class="w-1.5 h-1.5 rounded-full bg-red"></span>
                    closed
                  </span>
                {:else if s.is_idle}
                  <span class="inline-flex items-center gap-1 text-amber-400 text-xs">
                    <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                    {$tr('sessions.idling')}
                  </span>
                {:else}
                  <span class="inline-flex items-center gap-1 text-green text-xs">
                    <span class="w-1.5 h-1.5 rounded-full bg-green"></span>
                    {$tr('sessions.running')}
                  </span>
                {/if}
              </td>
              <td class="py-3 pr-4 font-mono tabular-nums text-dim text-xs">{fmtDuration(s.age_ms)}</td>
              <td class="py-3 pr-4 font-mono tabular-nums text-right text-xs text-text">{s.stream_count}</td>
              <td class="py-3 pr-5 font-mono tabular-nums text-right text-xs text-text">{s.packet_count.toLocaleString()}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
