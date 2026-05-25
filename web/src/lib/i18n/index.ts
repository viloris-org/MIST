import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';
import en from './en';
import zh from './zh';

export type Locale = 'en' | 'zh';

const translations: Record<Locale, Record<string, any>> = { en, zh };
const localeNames: Record<Locale, string> = { en: 'English', zh: '中文' };

function detectLocale(): Locale {
	if (!browser) return 'en';
	const stored = localStorage.getItem('mist_locale') as Locale | null;
	if (stored && translations[stored]) return stored;
	const nav = navigator.language.toLowerCase();
	if (nav.startsWith('zh')) return 'zh';
	return 'en';
}

export const locale = writable<Locale>(detectLocale());
export const localeName = derived(locale, $l => localeNames[$l]);
export const locales: Locale[] = ['en', 'zh'];

if (browser) {
	locale.subscribe(v => localStorage.setItem('mist_locale', v));
}

function get(obj: Record<string, any>, path: string): string {
	const keys = path.split('.');
	let cur: any = obj;
	for (const k of keys) {
		if (cur == null) return path;
		cur = cur[k];
	}
	return typeof cur === 'string' ? cur : path;
}

export function t(key: string): string {
	if (!browser) return key;
	const lang = localStorage.getItem('mist_locale') as Locale | null;
	const dict = translations[lang && translations[lang] ? lang : 'en'];
	return get(dict, key) ?? key;
}

// Reactive translation store — use as $t('nav.dashboard') in templates.
// Each component can import { t } from '$lib/i18n' for programmatic use,
// or use the $tr store for reactive re-renders when language changes.
export const tr = derived(locale, $l => {
	return (key: string): string => get(translations[$l], key) ?? key;
});
