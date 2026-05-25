<script lang="ts">
  import { status } from '$lib/stores/status';
  import { tr } from '$lib/i18n';
</script>

<div class="space-y-6">
  <h2 class="text-lg font-semibold">{$tr('logs.title')}</h2>

  <div class="card-sm">
    <h3 class="text-sm font-medium text-dim uppercase tracking-wider mb-4">{$tr('logs.serverError')}</h3>
    {#if $status?.last_error}
      <div class="bg-red/10 border border-red/30 rounded-md p-4">
        <pre class="text-red text-sm whitespace-pre-wrap font-mono">{$status.last_error}</pre>
      </div>
    {:else}
      <p class="text-dim text-sm">{$tr('logs.noErrors')}</p>
    {/if}
  </div>

  <div class="card-sm">
    <h3 class="text-sm font-medium text-dim uppercase tracking-wider mb-4">{$tr('logs.connectionInfo')}</h3>
    <dl class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
      <div>
        <dt class="text-dim">{$tr('logs.serverAddress')}</dt>
        <dd class="font-mono mt-0.5">{$status?.server ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim">{$tr('logs.version')}</dt>
        <dd class="font-mono mt-0.5">mist/{$status?.version ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim">{$tr('logs.startedAt')}</dt>
        <dd class="font-mono mt-0.5">{$status?.started_at ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim">{$tr('logs.lastUpdated')}</dt>
        <dd class="font-mono mt-0.5">{$status?.updated_at ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim">{$tr('logs.commit')}</dt>
        <dd class="font-mono mt-0.5">{$status?.commit?.slice(0, 8) ?? '--'}</dd>
      </div>
      <div>
        <dt class="text-dim">{$tr('logs.buildDate')}</dt>
        <dd class="font-mono mt-0.5">{$status?.date ?? '--'}</dd>
      </div>
    </dl>
  </div>
</div>
