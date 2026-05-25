<script lang="ts">
  import { serverState } from '$lib/stores/status';

  let { class: className = '' }: { class?: string } = $props();
</script>

<div class="flex items-center gap-2 {className}">
  <span
    class="relative flex h-2.5 w-2.5"
  >
    <span
      class="absolute inline-flex h-full w-full rounded-full opacity-75 animate-pulse-dot"
      class:bg-green={$serverState === 'running'}
      class:bg-red={$serverState !== 'running'}
    ></span>
    <span
      class="relative inline-flex rounded-full h-2.5 w-2.5"
      class:bg-green={$serverState === 'running'}
      class:bg-red={$serverState !== 'running'}
    ></span>
  </span>
  <span class="font-medium text-xs capitalize text-dim-light">{$serverState}</span>
</div>
