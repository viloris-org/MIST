<script lang="ts">
  import { status } from '$lib/stores/status';
  import { tr } from '$lib/i18n';
  import LogViewer from '$lib/components/LogViewer.svelte';
</script>

<div class="space-y-6 animate-fade-in">
  <div class="flex items-center justify-between">
    <h2 class="text-lg font-semibold text-text tracking-tight">{$tr('logs.title')}</h2>
  </div>

  <LogViewer />

  <div class="card-sm">
    <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('logs.connectionInfo')}</h3>
    <dl class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 text-sm">
      <div>
        <dt class="text-dim text-xs uppercase tracking-wider">{$tr('logs.serverAddress')}</dt>
        <dd class="font-mono mt-1 text-text">{$status?.server ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim text-xs uppercase tracking-wider">{$tr('logs.version')}</dt>
        <dd class="font-mono mt-1 text-text">mist/{$status?.version ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim text-xs uppercase tracking-wider">{$tr('logs.startedAt')}</dt>
        <dd class="font-mono mt-1 text-text text-xs">{$status?.started_at ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim text-xs uppercase tracking-wider">{$tr('logs.lastUpdated')}</dt>
        <dd class="font-mono mt-1 text-text text-xs">{$status?.updated_at ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim text-xs uppercase tracking-wider">{$tr('logs.commit')}</dt>
        <dd class="font-mono mt-1 text-text">{$status?.commit?.slice(0, 8) ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim text-xs uppercase tracking-wider">{$tr('logs.buildDate')}</dt>
        <dd class="font-mono mt-1 text-text">{$status?.date ?? '--'}</dd>
      </div>
    </dl>
  </div>

  {#if $status?.last_error}
    <div class="card-sm border-red/30">
      <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-3">{$tr('logs.serverError')}</h3>
      <div class="bg-red/5 border border-red/20 rounded-lg p-4">
        <pre class="text-red text-sm whitespace-pre-wrap font-mono leading-relaxed">{$status.last_error}</pre>
      </div>
    </div>
  {/if}
</div>
