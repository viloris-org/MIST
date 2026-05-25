<script lang="ts">
	import { onMount } from 'svelte';
	import { auth, isAuthenticated, isChecking } from '$lib/stores/auth';
	import { status, uptime, startPolling, stopPolling } from '$lib/stores/status';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { tr, locale, locales } from '$lib/i18n';
	import '../app.css';

	let { children } = $props();

	let authed = $derived($isAuthenticated);
	let checking = $derived($isChecking);
	let currentPath = $derived($page.url.pathname);

	onMount(() => {
		auth.check().then(ok => {
			if (!ok && currentPath !== '/login') {
				goto(`/login?from=${encodeURIComponent(currentPath)}`);
			}
		});

		const unsub = isAuthenticated.subscribe(v => {
			if (v) startPolling();
			else stopPolling();
		});

		return () => {
			unsub();
			stopPolling();
		};
	});

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}

	function isLoginPage(path: string) {
		return path.startsWith('/login');
	}
</script>

{#if $isChecking}
	<div class="min-h-screen flex items-center justify-center bg-bg">
		<div class="text-dim text-sm">{$tr('common.loading')}</div>
	</div>
{:else if isLoginPage($page.url.pathname)}
	{@render children()}
{:else if authed}
	<div class="min-h-screen flex flex-col">
		<!-- Header -->
		<header class="border-b border-border bg-surface/50 backdrop-blur">
			<div class="max-w-5xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
				<div class="flex items-center gap-6">
					<a href="/" class="text-lg font-semibold tracking-wide">MIST</a>
					<nav class="flex items-center gap-1">
						<a href="/" class="px-3 py-1.5 rounded text-sm text-dim hover:text-gray-100 hover:bg-surface transition-colors">
							{$tr('nav.dashboard')}
						</a>
						<a href="/sessions" class="px-3 py-1.5 rounded text-sm text-dim hover:text-gray-100 hover:bg-surface transition-colors">
							{$tr('nav.sessions')}
						</a>
						<a href="/logs" class="px-3 py-1.5 rounded text-sm text-dim hover:text-gray-100 hover:bg-surface transition-colors">
							{$tr('nav.logs')}
						</a>
					</nav>
				</div>
				<div class="flex items-center gap-4">
					<StatusBadge />
					<span class="text-sm text-dim tabular-nums hidden sm:inline">{$uptime}</span>

					<!-- Language switcher -->
					<div class="relative">
						<button
							onclick={() => locale.set($locale === 'en' ? 'zh' : 'en')}
							class="text-xs text-dim hover:text-gray-100 transition-colors cursor-pointer border border-border rounded px-1.5 py-0.5"
							title="{$tr('common.language')}"
						>
							{$locale === 'en' ? 'EN' : '中'}
						</button>
					</div>

					<button
						onclick={handleLogout}
						class="text-sm text-dim hover:text-gray-100 transition-colors cursor-pointer"
						title="{$tr('common.logout')}"
					>
						<svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
							<polyline points="16 17 21 12 16 7" />
							<line x1="21" y1="12" x2="9" y2="12" />
						</svg>
					</button>
				</div>
			</div>
		</header>

		<main class="flex-1 max-w-5xl mx-auto px-4 sm:px-6 py-6 w-full">
			{@render children()}
		</main>

		<footer class="border-t border-border py-4 text-center">
			<span class="text-xs text-dim">mist/{$status?.version ?? '--'}</span>
		</footer>
	</div>
{:else}
	<div class="min-h-screen flex items-center justify-center bg-bg">
		<div class="text-dim text-sm">{$tr('common.redirecting')}</div>
	</div>
{/if}
