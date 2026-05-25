<script lang="ts">
  import type { DNSStats } from '$lib/types';
  import { tr } from '$lib/i18n';
  import StatCard from './StatCard.svelte';

  let { stats }: { stats: DNSStats } = $props();
</script>

<section>
  <h2 class="text-sm font-medium text-dim uppercase tracking-wider mb-3">{$tr('dashboard.dnsProxy')}</h2>
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
    <StatCard label={$tr('dashboard.forwarded')} value={stats.queries_forwarded} />
    <StatCard label={$tr('dashboard.cached')} value={stats.queries_cached} />
    <StatCard label={$tr('dashboard.failed')} value={stats.queries_failed} />
    <StatCard label={$tr('dashboard.cacheSize')} value={stats.cache_size} />
  </div>
  {#if stats.upstreams.length > 0}
    <div class="text-xs text-dim mt-2 font-mono">
      {$tr('dashboard.upstreams')}: {stats.upstreams.join(', ')}
    </div>
  {/if}
</section>
