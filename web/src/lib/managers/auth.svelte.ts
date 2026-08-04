import { getCurrentUser, login, logout, setup, type UserDto } from '$lib/sdk';

class AuthManager {
	user = $state<UserDto | null>(null);
	private loaded = false;

	async load(): Promise<UserDto | null> {
		if (!this.loaded) {
			try {
				this.user = await getCurrentUser();
			} catch {
				this.user = null;
			}
			this.loaded = true;
		}
		return this.user;
	}

	async login(username: string, password: string) {
		this.user = await login({ username, password });
		this.loaded = true;
	}

	async setup(username: string, password: string) {
		this.user = await setup({ username, password, isAdmin: true });
		this.loaded = true;
	}

	async logout() {
		try {
			await logout();
		} finally {
			this.user = null;
			this.loaded = true;
		}
	}
}

export const auth = new AuthManager();
