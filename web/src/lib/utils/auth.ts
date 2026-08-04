import { goto } from '$app/navigation';
import { auth } from '$lib/managers/auth.svelte';
import { redirect } from '@sveltejs/kit';

export async function authenticate(options?: { admin?: boolean }) {
	const user = await auth.load();
	if (!user) redirect(302, '/login');
	if (options?.admin && !user.isAdmin) redirect(302, '/');
	return user;
}

export async function requireGuest() {
	if (await auth.load()) redirect(302, '/');
}

export async function logoutAndRedirect() {
	await auth.logout();
	await goto('/login');
}
