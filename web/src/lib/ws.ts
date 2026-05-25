import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { auth } from '$lib/stores/auth';
import type { StatusResponse } from '$lib/types';

export type WSMessageType = 'status' | 'traffic' | 'log';

export interface WSMessage {
	type: WSMessageType;
	payload: unknown;
	time: number;
}

type Listener = (msg: WSMessage) => void;

const listeners = new Map<WSMessageType, Set<Listener>>();
let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelay = 1000;
const maxReconnectDelay = 30000;
let running = false;

export const wsConnected = writable(false);

function connect() {
	if (!browser || !running || ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) return;

	const token = browser ? localStorage.getItem('mist_token') : null;
	if (!token) return;

	const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
	const url = `${proto}//${location.host}/api/ws?token=${encodeURIComponent(token)}`;

	ws = new WebSocket(url);
	ws.onopen = () => {
		reconnectDelay = 1000;
		wsConnected.set(true);
	};

	ws.onmessage = (event) => {
		try {
			const msg: WSMessage = JSON.parse(event.data);
			const subs = listeners.get(msg.type);
			if (subs) {
				for (const fn of subs) fn(msg);
			}
		} catch { /* ignore malformed messages */ }
	};

	ws.onclose = () => {
		wsConnected.set(false);
		ws = null;
		scheduleReconnect();
	};

	ws.onerror = () => {
		ws?.close();
	};
}

function scheduleReconnect() {
	if (!running) return;
	reconnectTimer = setTimeout(() => {
		reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
		connect();
	}, reconnectDelay + Math.random() * 1000);
}

export function startWS() {
	if (running) return;
	running = true;
	connect();
}

export function stopWS() {
	running = false;
	if (reconnectTimer) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
	if (ws) {
		ws.onclose = null;
		ws.close();
		ws = null;
	}
	wsConnected.set(false);
}

export function onWS(type: WSMessageType, fn: Listener): () => void {
	if (!listeners.has(type)) listeners.set(type, new Set());
	listeners.get(type)!.add(fn);
	return () => {
		const subs = listeners.get(type);
		if (subs) {
			subs.delete(fn);
			if (subs.size === 0) listeners.delete(type);
		}
	};
}
