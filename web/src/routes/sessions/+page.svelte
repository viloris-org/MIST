<script lang="ts">
  import { status } from '$lib/stores/status';
  import { tr } from '$lib/i18n';
  import StatCard from '$lib/components/StatCard.svelte';
  import SessionTable from '$lib/components/SessionTable.svelte';
</script>

<div class="space-y-6 animate-fade-in">
  <div class="flex items-center justify-between">
    <h2 class="text-lg font-semibold text-text tracking-tight">{$tr('sessions.title')}</h2>
  </div>

  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
    <StatCard
      label={$tr('sessions.activeSessions')}
      value={$status?.client?.session_pool?.sessions ?? 0}
    />
    <StatCard
      label={$tr('sessions.idleSessions')}
      value={$status?.client?.session_pool?.idle_sessions ?? 0}
    />
    <StatCard
      label={$tr('sessions.openStreams')}
      value={$status?.client?.session_pool?.open_streams ?? 0}
    />
    <StatCard
      label={$tr('sessions.cumulativeStreams')}
      value={$status?.client?.session_pool?.cumulative_streams ?? 0}
    />
  </div>

  <SessionTable sessions={$status?.client?.sessions ?? []} />

  <div class="card-sm">
    <h3 class="text-sm font-medium text-dim uppercase tracking-wider mb-4">{$tr('sessions.connections')}</h3>
    <div class="grid grid-cols-2 gap-6 text-sm">
      <div>
        <span class="text-dim text-xs uppercase tracking-wider">{$tr('sessions.activeConnections')}</span>
        <div class="text-2xl font-semibold tabular-nums mt-1.5 font-mono text-text">
          {$status?.active_connections ?? 0}
        </div>
      </div>
      <div>
        <span class="text-dim text-xs uppercase tracking-wider">{$tr('sessions.acceptedCumulative')}</span>
        <div class="text-2xl font-semibold tabular-nums mt-1.5 font-mono text-text">
          {$status?.accepted_connections ?? 0}
        </div>
      </div>
    </div>
  </div>
</div>
