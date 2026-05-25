<script lang="ts">
  import type { TUNStats } from '$lib/types';
  import { tr } from '$lib/i18n';
  import StatCard from './StatCard.svelte';

  let { stats }: { stats: TUNStats } = $props();
</script>

<section>
  <h2 class="text-sm font-medium text-dim uppercase tracking-wider mb-3">{$tr('dashboard.tunInterface')}</h2>
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
    <StatCard label={$tr('dashboard.interface')} value={stats.name} subtitle={stats.address} />
    <StatCard label={$tr('dashboard.rxPackets')} value={stats.rx_packets} />
    <StatCard label={$tr('dashboard.txPackets')} value={stats.tx_packets} />
    <StatCard label={$tr('dashboard.mtu')} value={stats.mtu} />
  </div>
  <div class="grid grid-cols-2 gap-3 mt-3">
    <StatCard label={$tr('dashboard.rxData')} value={stats.rx_bytes} format="bytes" />
    <StatCard label={$tr('dashboard.txData')} value={stats.tx_bytes} format="bytes" />
  </div>
</section>
