import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

export interface AuthState {
	token: string | null;
	authenticated: boolean;
	noPasswordSet: boolean;
	checking: boolean;
}

function createAuthStore() {
	const stored = browser ? localStorage.getItem('mist_token') : null;
	const store = writable<AuthState>({
		token: stored,
		authenticated: false,
		noPasswordSet: false,
		checking: true
	});

	const { subscribe, set, update } = store;

	async function check(): Promise<boolean> {
		try {
			const resp = await fetch('/api/check', {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});
			const body = await resp.json();

			if (body.no_password_set) {
				update(s => ({ ...s, authenticated: true, noPasswordSet: true, checking: false }));
				return true;
			}

			if (body.authenticated) {
				update(s => ({ ...s, authenticated: true, checking: false }));
				return true;
			}

			update(s => ({ ...s, authenticated: false, token: null, checking: false }));
			if (browser) localStorage.removeItem('mist_token');
			return false;
		} catch {
			update(s => ({ ...s, checking: false }));
			return false;
		}
	}

	let token: string | null = stored;
	subscribe(s => { token = s.token; });

	return {
		subscribe,
		check,
		async login(password: string): Promise<string | null> {
			try {
				const resp = await fetch('/api/login', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ password })
				});
				const body = await resp.json();
				if (!resp.ok) return body.error || 'Login failed';
				if (browser) localStorage.setItem('mist_token', body.token);
				set({ token: body.token, authenticated: true, noPasswordSet: false, checking: false });
				return null;
			} catch (e) {
				return e instanceof Error ? e.message : 'Network error';
			}
		},
		async logout() {
			try {
				await fetch('/api/logout', {
					method: 'POST',
					headers: token ? { Authorization: `Bearer ${token}` } : {}
				});
			} catch { /* ignore */ }
			if (browser) localStorage.removeItem('mist_token');
			set({ token: null, authenticated: false, noPasswordSet: false, checking: false });
		}
	};
}

export const auth = createAuthStore();
export const isAuthenticated = derived(auth, $a => $a.authenticated);
export const isChecking = derived(auth, $a => $a.checking);
