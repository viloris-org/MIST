<script lang="ts">
	import { auth } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { tr, locale } from '$lib/i18n';

	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!password) return;
		loading = true;
		error = '';
		const err = await auth.login(password);
		if (err) {
			error = err;
			loading = false;
		} else {
			const from = $page.url.searchParams.get('from') || '/';
			goto(from);
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-bg px-4">
	<div class="w-full max-w-sm">
		<div class="text-center mb-10">
			<h1 class="text-3xl font-bold tracking-tight font-mono text-text">
				<span class="text-accent">M</span>IST
			</h1>
			<p class="text-dim text-sm mt-2">{$tr('login.subtitle')}</p>
		</div>

		<form onsubmit={handleSubmit} class="card-elevated space-y-4">
			<div>
				<label for="password" class="block text-xs text-dim uppercase tracking-wider mb-2 font-medium">{$tr('login.password')}</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					autocomplete="current-password"
					class="w-full bg-bg border border-border rounded-lg px-3.5 py-2.5 text-sm text-text font-mono
					       focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent/30 transition-all duration-200
					       placeholder:text-dim/50"
					placeholder={$tr('login.placeholder')}
				/>
			</div>

			{#if error}
				<div class="text-red text-sm bg-red/5 border border-red/20 rounded-lg px-3 py-2">{error}</div>
			{/if}

			<button
				type="submit"
				disabled={loading || !password}
				class="w-full py-2.5 rounded-lg text-sm font-semibold transition-all duration-200 cursor-pointer
				       bg-accent text-bg hover:shadow-glow-sm hover:shadow-accent/30
				       disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:shadow-none"
			>
				{loading ? $tr('login.signingIn') : $tr('login.signIn')}
			</button>
		</form>

		<div class="mt-8 flex justify-center gap-2">
			<button
				onclick={() => locale.set('en')}
				class="text-xs px-3 py-1.5 rounded-md border font-medium transition-all duration-200 cursor-pointer
				       {$locale === 'en'
						? 'border-accent bg-accent/10 text-accent'
						: 'border-border text-dim hover:text-dim-light hover:border-border-light'}"
			>English</button>
			<button
				onclick={() => locale.set('zh')}
				class="text-xs px-3 py-1.5 rounded-md border font-medium transition-all duration-200 cursor-pointer
				       {$locale === 'zh'
						? 'border-accent bg-accent/10 text-accent'
						: 'border-border text-dim hover:text-dim-light hover:border-border-light'}"
			>中文</button>
		</div>
	</div>
</div>
