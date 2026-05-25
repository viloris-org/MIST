<script lang="ts">
  import { status } from '$lib/stores/status';
  import { tr } from '$lib/i18n';
  import StatCard from '$lib/components/StatCard.svelte';
  import TrafficChart from '$lib/components/TrafficChart.svelte';
  import TUNStats from '$lib/components/TUNStats.svelte';
  import DNSStats from '$lib/components/DNSStats.svelte';

  let traffic: TrafficChart | undefined = $state();

  $effect(() => {
    if ($status?.tun && traffic) {
      traffic.push($status);
    }
  });
</script>

<div class="space-y-6">
  <!-- Session stats grid -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
    <StatCard
      label={$tr('dashboard.activeSessions')}
      value={$status?.client?.session_pool?.sessions ?? 0}
      subtitle="{($status?.client?.session_pool?.idle_sessions ?? 0)} {$tr('dashboard.idle')}"
    />
    <StatCard
      label={$tr('dashboard.openStreams')}
      value={$status?.client?.session_pool?.open_streams ?? 0}
      subtitle="{($status?.client?.session_pool?.cumulative_streams ?? 0)} {$tr('dashboard.total')}"
    />
    <StatCard
      label={$tr('dashboard.connections')}
      value={$status?.active_connections ?? 0}
      subtitle="{($status?.accepted_connections ?? 0)} {$tr('dashboard.accepted')}"
    />
    <StatCard
      label={$tr('dashboard.totalSessions')}
      value={$status?.client?.session_pool?.cumulative_sessions ?? 0}
      subtitle={$tr('dashboard.sinceStart')}
    />
  </div>

  <!-- Server info bar -->
  <div class="card-sm flex flex-wrap items-center justify-between gap-3 text-sm">
    <div class="flex items-center gap-4">
      <span class="text-dim">{$tr('dashboard.server')}</span>
      <span class="font-mono text-sm">{$status?.server ?? '--'}</span>
    </div>
    <div class="flex items-center gap-4">
      <span class="text-dim">{$tr('dashboard.version')}</span>
      <span class="font-mono text-sm">mist/{$status?.version ?? '--'}</span>
    </div>
    {#if $status?.last_error}
      <div class="text-red text-sm w-full">{$status.last_error}</div>
    {/if}
  </div>

  <!-- Traffic chart (only when TUN is active) -->
  {#if $status?.tun}
    <TrafficChart bind:this={traffic} />
  {/if}

  <!-- TUN stats -->
  {#if $status?.tun}
    <TUNStats stats={$status.tun} />
  {/if}

  <!-- DNS stats -->
  {#if $status?.dns}
    <DNSStats stats={$status.dns} />
  {/if}
</div>
