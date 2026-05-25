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

	function isActive(path: string, target: string) {
		if (target === '/') return path === '/';
		return path.startsWith(target);
	}
</script>

{#if $isChecking}
	<div class="min-h-screen flex items-center justify-center bg-bg">
		<div class="text-dim text-sm animate-pulse">{$tr('common.loading')}</div>
	</div>
{:else if isLoginPage($page.url.pathname)}
	{@render children()}
{:else if authed}
	<div class="min-h-screen flex flex-col bg-bg">
		<!-- Header -->
		<header class="sticky top-0 z-50 border-b border-border/50 bg-surface/80 backdrop-blur-lg">
			<div class="max-w-6xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
				<div class="flex items-center gap-8">
					<a href="/" class="text-lg font-semibold tracking-tight font-mono text-text">
						<span class="text-accent">M</span>IST
					</a>
					<nav class="hidden sm:flex items-center gap-0.5">
						{#each [
							['/', 'nav.dashboard'],
							['/sessions', 'nav.sessions'],
							['/logs', 'nav.logs'],
							['/settings', 'nav.settings']
						] as [href, label]}
							<a
								{href}
								class="relative px-3 py-1.5 rounded-lg text-sm transition-colors duration-200 cursor-pointer
									{isActive(currentPath, href)
										? 'text-text bg-accent/10'
										: 'text-dim hover:text-dim-light hover:bg-surface-alt'}"
							>
								{$tr(label)}
								{#if isActive(currentPath, href)}
									<span class="absolute bottom-0 left-1/2 -translate-x-1/2 w-4 h-0.5 bg-accent rounded-full"></span>
								{/if}
							</a>
						{/each}
					</nav>
				</div>

				<div class="flex items-center gap-3">
					<div class="hidden sm:flex items-center gap-3">
						<StatusBadge />
						<span class="text-xs text-dim tabular-nums font-mono">{$uptime}</span>
					</div>

					<!-- Mobile nav -->
					<div class="sm:hidden flex items-center gap-1">
						{#each [
							['/', 'nav.dashboard'],
							['/sessions', 'nav.sessions'],
							['/logs', 'nav.logs'],
							['/settings', 'nav.settings']
						] as [href, label]}
							<a
								{href}
								class="px-2 py-1 rounded text-xs transition-colors duration-200 cursor-pointer
									{isActive(currentPath, href)
										? 'text-text bg-accent/10'
										: 'text-dim hover:text-dim-light'}"
							>
								{$tr(label)}
							</a>
						{/each}
					</div>

					<div class="h-5 w-px bg-border hidden sm:block"></div>

					<button
						onclick={() => locale.set($locale === 'en' ? 'zh' : 'en')}
						class="text-xs font-medium text-dim hover:text-text transition-colors duration-200 cursor-pointer
							border border-border hover:border-border-light rounded-md px-1.5 py-0.5"
					>
						{$locale === 'en' ? 'EN' : '中'}
					</button>

					<button
						onclick={handleLogout}
						class="text-dim hover:text-red transition-colors duration-200 cursor-pointer p-1"
						title={$tr('common.logout')}
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

		<main class="flex-1 max-w-6xl mx-auto px-4 sm:px-6 py-8 w-full animate-fade-in">
			{@render children()}
		</main>

		<footer class="border-t border-border/50 py-4 text-center">
			<span class="text-xs text-dim font-mono">mist/{$status?.version ?? '--'}</span>
		</footer>
	</div>
{:else}
	<div class="min-h-screen flex items-center justify-center bg-bg">
		<div class="text-dim text-sm">{$tr('common.redirecting')}</div>
	</div>
{/if}
