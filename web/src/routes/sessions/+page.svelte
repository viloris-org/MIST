<script lang="ts">
  import { status } from '$lib/stores/status';
  import { tr } from '$lib/i18n';
  import StatCard from '$lib/components/StatCard.svelte';
</script>

<div class="space-y-6">
  <h2 class="text-lg font-semibold">{$tr('sessions.title')}</h2>

  <!-- Summary cards -->
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

  <!-- Session pool info -->
  <div class="card-sm">
    <h3 class="text-sm font-medium text-dim uppercase tracking-wider mb-4">{$tr('sessions.sessionPool')}</h3>
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
      <div>
        <span class="text-dim">{$tr('sessions.active')}</span>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {$status?.client?.session_pool?.sessions ?? 0}
        </div>
      </div>
      <div>
        <span class="text-dim">{$tr('sessions.idle')}</span>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {$status?.client?.session_pool?.idle_sessions ?? 0}
        </div>
      </div>
      <div>
        <span class="text-dim">{$tr('sessions.openStreams')}</span>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {$status?.client?.session_pool?.open_streams ?? 0}
        </div>
      </div>
      <div>
        <span class="text-dim">{$tr('sessions.cumulative')}</span>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {$status?.client?.session_pool?.cumulative_sessions ?? 0}
        </div>
      </div>
    </div>
  </div>

  <!-- Connections info -->
  <div class="card-sm">
    <h3 class="text-sm font-medium text-dim uppercase tracking-wider mb-4">{$tr('sessions.connections')}</h3>
    <div class="grid grid-cols-2 gap-4 text-sm">
      <div>
        <span class="text-dim">{$tr('sessions.activeConnections')}</span>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {$status?.active_connections ?? 0}
        </div>
      </div>
      <div>
        <span class="text-dim">{$tr('sessions.acceptedCumulative')}</span>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {$status?.accepted_connections ?? 0}
        </div>
      </div>
    </div>
  </div>
</div>
