<script lang="ts">
  import { onMount } from 'svelte';
  import { tr, locale, locales } from '$lib/i18n';
  import { fetchConfig } from '$lib/api';
  import type { RuntimeConfig } from '$lib/types';

  let config: RuntimeConfig | null = $state(null);
  let activeTab = $state('general');
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      config = await fetchConfig();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load config';
    } finally {
      loading = false;
    }
  });

  const tabs = [
    { key: 'general', label: () => $tr('settings.general') },
    { key: 'appearance', label: () => $tr('settings.appearance') },
  ];

  function v(val: any, mono = false) {
    return val === undefined || val === null ? '--' : String(val);
  }
</script>

<div class="space-y-6 animate-fade-in">
  <h2 class="text-lg font-semibold text-text tracking-tight">{$tr('settings.title')}</h2>

  <div class="flex border-b border-border">
    {#each tabs as tab}
      <button
        onclick={() => activeTab = tab.key}
        class="relative px-4 py-2.5 text-sm font-medium transition-colors duration-200 cursor-pointer
          {activeTab === tab.key ? 'text-text' : 'text-dim hover:text-dim-light'}"
      >
        {tab.label()}
        {#if activeTab === tab.key}
          <span class="absolute bottom-0 left-0 right-0 h-0.5 bg-accent rounded-full"></span>
        {/if}
      </button>
    {/each}
  </div>

  {#if loading}
    <div class="text-dim text-sm py-12 text-center">{$tr('common.loading')}</div>
  {:else if error}
    <div class="bg-red/5 border border-red/20 rounded-lg p-4 text-red text-sm">{error}</div>
  {:else if config}
    {#if activeTab === 'general'}
      <div class="space-y-4">
        <div class="card-sm">
          <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('settings.connection')}</h3>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.server')}</dt><dd class="mt-1 font-mono text-text">{v(config.server)}</dd></div>
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.listenAddr')}</dt><dd class="mt-1 font-mono text-text">{v(config.listen)}</dd></div>
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.inbound')}</dt><dd class="mt-1 font-mono text-text">{v(config.inbound)}</dd></div>
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.redirectListen')}</dt><dd class="mt-1 font-mono text-text">{v(config.redirect_listen)}</dd></div>
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.minIdleSession')}</dt><dd class="mt-1 font-mono text-text">{v(config.min_idle_session)}</dd></div>
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.tlsMinVersion')}</dt><dd class="mt-1 font-mono text-text">{v(config.tls_min_version)}</dd></div>
            <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.insecure')}</dt><dd class="mt-1 font-mono text-text">{v(config.insecure)}</dd></div>
          </dl>
        </div>

        {#if config.tun}
          <div class="card-sm">
            <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('dashboard.tunInterface')}</h3>
            <dl class="grid grid-cols-1 sm:grid-cols-3 gap-x-8 gap-y-3 text-sm">
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('dashboard.interface')}</dt><dd class="mt-1 font-mono text-text">{v(config.tun.name)}</dd></div>
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('dashboard.mtu')}</dt><dd class="mt-1 font-mono text-text">{v(config.tun.mtu)}</dd></div>
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.address')}</dt><dd class="mt-1 font-mono text-text">{v(config.tun.address)}</dd></div>
            </dl>
          </div>
        {/if}

        {#if config.dns}
          <div class="card-sm">
            <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('dashboard.dnsProxy')}</h3>
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.listenAddr')}</dt><dd class="mt-1 font-mono text-text">{v(config.dns.listen)}</dd></div>
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('dashboard.upstreams')}</dt><dd class="mt-1 font-mono text-text">{v(config.dns.upstream)}</dd></div>
            </dl>
          </div>
        {/if}

        {#if config.web}
          <div class="card-sm">
            <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('settings.webDashboard')}</h3>
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.listenAddr')}</dt><dd class="mt-1 font-mono text-text">{v(config.web.listen)}</dd></div>
              <div><dt class="text-dim text-xs uppercase tracking-wider">{$tr('settings.password')}</dt><dd class="mt-1 text-text">{config.web.has_password ? 'configured' : $tr('settings.notSet')}</dd></div>
            </dl>
          </div>
        {/if}

        <p class="text-xs text-dim mt-6">{$tr('settings.configHint')}</p>
      </div>

    {:else if activeTab === 'appearance'}
      <div class="card-sm space-y-4">
        <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('settings.language')}</h3>
        <div class="flex gap-2">
          {#each locales as l}
            <button
              onclick={() => locale.set(l)}
              class="px-4 py-2 rounded-lg border text-sm font-medium cursor-pointer transition-all duration-200
                {$locale === l
                  ? 'border-accent bg-accent/10 text-accent'
                  : 'border-border text-dim hover:text-dim-light hover:border-border-light'}"
            >
              {l === 'en' ? 'English' : '中文'}
            </button>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>
