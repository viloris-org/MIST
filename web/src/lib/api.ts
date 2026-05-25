import type { StatusResponse, RuntimeConfig } from '$lib/types';
import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';

function authHeaders(): Record<string, string> {
	const { token } = get(auth);
	if (token) return { Authorization: `Bearer ${token}` };
	return {};
}

export async function fetchStatus(): Promise<StatusResponse> {
	const resp = await fetch('/api/status', { headers: authHeaders() });
	if (!resp.ok) {
		if (resp.status === 401) {
			auth.logout();
		}
		throw new Error(`HTTP ${resp.status}`);
	}
	return resp.json();
}

export async function fetchConfig(): Promise<RuntimeConfig> {
	const resp = await fetch('/api/config', { headers: authHeaders() });
	if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
	return resp.json();
}
