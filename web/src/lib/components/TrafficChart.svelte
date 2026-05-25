<script lang="ts">
  import { onMount } from 'svelte';
  import { Chart, registerables } from 'chart.js';
  import type { StatusResponse } from '$lib/types';
  import { tr } from '$lib/i18n';

  Chart.register(...registerables);

  let canvas: HTMLCanvasElement;
  let chart: Chart | null = null;

  const MAX_POINTS = 60;
  let rxHistory: number[] = [];
  let txHistory: number[] = [];
  let prevRx = 0;
  let prevTx = 0;
  let prevTime = 0;

  export function push(status: StatusResponse) {
    if (!status.tun) return;

    const now = Date.now();
    let rxRate = 0;
    let txRate = 0;

    if (prevTime > 0 && now > prevTime) {
      const dt = (now - prevTime) / 1000;
      rxRate = Math.max(0, (status.tun.rx_bytes - prevRx) / dt);
      txRate = Math.max(0, (status.tun.tx_bytes - prevTx) / dt);
    }

    prevRx = status.tun.rx_bytes;
    prevTx = status.tun.tx_bytes;
    prevTime = now;

    rxHistory.push(rxRate);
    txHistory.push(txRate);
    if (rxHistory.length > MAX_POINTS) rxHistory.shift();
    if (txHistory.length > MAX_POINTS) txHistory.shift();

    updateChart();
  }

  function updateChart() {
    if (!chart) return;
    chart.data.labels = rxHistory.map((_, i) => i);
    chart.data.datasets[0].data = rxHistory;
    chart.data.datasets[1].data = txHistory;
    chart.update('none');
  }

  onMount(() => {
    chart = new Chart(canvas, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: 'RX',
            data: [],
            borderColor: '#38BDF8',
            backgroundColor: 'rgba(56,189,248,0.08)',
            fill: true,
            tension: 0.4,
            pointRadius: 0,
            borderWidth: 1.5,
          },
          {
            label: 'TX',
            data: [],
            borderColor: '#22C55E',
            backgroundColor: 'rgba(34,197,94,0.08)',
            fill: true,
            tension: 0.4,
            pointRadius: 0,
            borderWidth: 1.5,
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: { duration: 0 },
        interaction: {
          intersect: false,
          mode: 'index',
        },
        scales: {
          x: {
            display: true,
            ticks: { color: '#64748B', maxTicksLimit: 6, font: { size: 10 } },
            grid: { color: '#1E293B' }
          },
          y: {
            display: true,
            ticks: {
              color: '#64748B',
              font: { size: 10 },
              callback: (v) => fmtBytes(Number(v)) + '/s'
            },
            grid: { color: '#1E293B' },
            beginAtZero: true
          }
        },
        plugins: {
          legend: {
            labels: {
              color: '#94A3B8',
              usePointStyle: true,
              pointStyleWidth: 8,
              font: { size: 11 },
              padding: 16,
            }
          },
          tooltip: {
            backgroundColor: '#0F172A',
            borderColor: '#334155',
            borderWidth: 1,
            titleColor: '#F8FAFC',
            bodyColor: '#94A3B8',
            padding: 10,
          }
        }
      }
    });
  });

  function fmtBytes(n: number): string {
    if (n >= 1e9) return (n / 1e9).toFixed(1) + ' GB';
    if (n >= 1e6) return (n / 1e6).toFixed(1) + ' MB';
    if (n >= 1e3) return (n / 1e3).toFixed(1) + ' KB';
    return Math.round(n) + ' B';
  }
</script>

<div class="card-elevated">
  <h3 class="text-xs font-medium text-dim uppercase tracking-wider mb-4">{$tr('dashboard.traffic')}</h3>
  <div class="h-52">
    <canvas bind:this={canvas}></canvas>
  </div>
</div>
