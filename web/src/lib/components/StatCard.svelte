<script lang="ts">
  import { tweened } from 'svelte/motion';

  interface Props {
    label: string;
    value?: number | string;
    subtitle?: string;
    format?: 'number' | 'bytes';
    class?: string;
  }

  let { label, value = 0, subtitle = '', format = 'number', class: className = '' }: Props = $props();
  let animated = tweened(typeof value === 'number' ? value : 0, { duration: 400, easing: x => 1 - Math.pow(1 - x, 3) });

  $effect(() => {
    if (typeof value === 'number') animated.set(value);
  });

  function fmt(v: number | string): string {
    if (typeof v === 'string') return v;
    if (format === 'bytes') return fmtBytes(v);
    if (v >= 1e9) return (v / 1e9).toFixed(1) + 'G';
    if (v >= 1e6) return (v / 1e6).toFixed(1) + 'M';
    if (v >= 1e3) return (v / 1e3).toFixed(1) + 'K';
    return v.toString();
  }

  function fmtBytes(n: number): string {
    if (n >= 1e9) return (n / 1e9).toFixed(2) + ' GB';
    if (n >= 1e6) return (n / 1e6).toFixed(2) + ' MB';
    if (n >= 1e3) return (n / 1e3).toFixed(2) + ' KB';
    return n + ' B';
  }
</script>

<div class="card group cursor-pointer {className}">
  <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-2.5 transition-colors duration-200 group-hover:text-dim-light">{label}</h3>
  <div class="text-2xl font-semibold tabular-nums font-mono tracking-tight text-text">
    {typeof value === 'number' ? fmt(Math.round($animated)) : value}
  </div>
  {#if subtitle}
    <div class="text-xs text-dim mt-1.5 font-medium">{subtitle}</div>
  {/if}
</div>
