import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'system';

const KEY = 'dropla-theme';

function systemPrefersDark() {
	return browser && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

class ThemeManager {
	preference = $state<Theme>('system');

	constructor() {
		if (!browser) return;

		const stored = localStorage.getItem(KEY);
		if (stored === 'light' || stored === 'dark' || stored === 'system') {
			this.preference = stored;
		}

		// Follow the OS while the preference is "system", so the app changes with
		// it rather than only at load.
		window
			.matchMedia('(prefers-color-scheme: dark)')
			.addEventListener('change', () => this.apply());

		this.apply();
	}

	get resolved(): 'light' | 'dark' {
		if (this.preference === 'system') return systemPrefersDark() ? 'dark' : 'light';
		return this.preference;
	}

	set(preference: Theme) {
		this.preference = preference;
		if (browser) localStorage.setItem(KEY, preference);
		this.apply();
	}

	toggle() {
		this.set(this.resolved === 'dark' ? 'light' : 'dark');
	}

	private apply() {
		if (!browser) return;
		document.documentElement.classList.toggle('dark', this.resolved === 'dark');
		document.documentElement.style.colorScheme = this.resolved;
	}
}

export const theme = new ThemeManager();
