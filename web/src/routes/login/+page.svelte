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
		<div class="text-center mb-8">
			<h1 class="text-2xl font-semibold tracking-wide">{$tr('login.title')}</h1>
			<p class="text-dim text-sm mt-2">{$tr('login.subtitle')}</p>
		</div>

		<form onsubmit={handleSubmit} class="card space-y-4">
			<div>
				<label for="password" class="block text-sm text-dim mb-1.5">{$tr('login.password')}</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					autocomplete="current-password"
					class="w-full bg-bg border border-border rounded-md px-3 py-2 text-sm
					       focus:outline-none focus:border-accent transition-colors
					       placeholder:text-dim/60"
					placeholder={$tr('login.placeholder')}
				/>
			</div>

			{#if error}
				<div class="text-red text-sm">{error}</div>
			{/if}

			<button
				type="submit"
				disabled={loading || !password}
				class="w-full py-2 rounded-md text-sm font-medium transition-colors
				       bg-accent text-white hover:bg-accent/90
				       disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{loading ? $tr('login.signingIn') : $tr('login.signIn')}
			</button>
		</form>

		<!-- Language switcher on login page -->
		<div class="mt-6 flex justify-center gap-2">
			<button
				onclick={() => locale.set('en')}
				class="text-xs px-2 py-1 rounded border transition-colors cursor-pointer
				       {$locale === 'en'
						? 'border-accent text-accent'
						: 'border-border text-dim hover:text-gray-100'}"
			>English</button>
			<button
				onclick={() => locale.set('zh')}
				class="text-xs px-2 py-1 rounded border transition-colors cursor-pointer
				       {$locale === 'zh'
						? 'border-accent text-accent'
						: 'border-border text-dim hover:text-gray-100'}"
			>中文</button>
		</div>
	</div>
</div>
