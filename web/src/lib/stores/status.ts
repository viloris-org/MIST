import { writable, derived } from 'svelte/store';
import type { StatusResponse } from '$lib/types';
import { fetchStatus } from '$lib/api';
import { startWS, stopWS, onWS, wsConnected } from '$lib/ws';

export const status = writable<StatusResponse | null>(null);
export const statusError = writable<string | null>(null);

let timer: ReturnType<typeof setInterval> | null = null;
let unsubWS: (() => void) | null = null;

function tick() {
	fetchStatus().then(s => {
		status.set(s);
		statusError.set(null);
	}).catch(e => {
		statusError.set(e.message);
	});
}

export function startPolling(ms = 5000) {
	if (timer) return;

	// Start WebSocket for real-time push.
	startWS();

	// Receive status from WebSocket directly.
	unsubWS = onWS('status', (msg) => {
		status.set(msg.payload as StatusResponse);
		statusError.set(null);
	});

	// Polling as fallback (every 5s instead of 2s when WS is connected).
	tick();
	timer = setInterval(tick, ms);
}

export function stopPolling() {
	stopWS();
	if (unsubWS) {
		unsubWS();
		unsubWS = null;
	}
	if (timer) {
		clearInterval(timer);
		timer = null;
	}
}

// Derived stores
export const serverState = derived(status, $s => $s?.server_state ?? 'unknown');
export const uptime = derived(status, $s => {
	if (!$s?.started_at) return '';
	const elapsed = Math.floor((Date.now() - new Date($s.started_at).getTime()) / 1000);
	const d = Math.floor(elapsed / 86400);
	const h = Math.floor((elapsed % 86400) / 3600);
	const m = Math.floor((elapsed % 3600) / 60);
	const s = elapsed % 60;
	const parts: string[] = [];
	if (d > 0) parts.push(`${d}d`);
	if (h > 0) parts.push(`${h}h`);
	if (m > 0) parts.push(`${m}m`);
	if (parts.length === 0 || s > 0) parts.push(`${s}s`);
	return `up ${parts.join(' ') || '0s'}`;
});
