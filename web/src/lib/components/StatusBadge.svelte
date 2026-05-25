<script lang="ts">
  import { serverState } from '$lib/stores/status';

  let { class: className = '' }: { class?: string } = $props();
</script>

<div class="flex items-center gap-2 {className}">
  <span
    class="inline-block w-2 h-2 rounded-full"
    class:bg-green={$serverState === 'running'}
    class:bg-red={$serverState !== 'running'}
  ></span>
  <span class="font-medium text-sm capitalize">{$serverState}</span>
</div>
