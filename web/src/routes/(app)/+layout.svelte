<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '$lib/components/ui/button';
	import { Toaster } from '$lib/components/ui/sonner';
	import { theme } from '$lib/theme.svelte';
	import { logoutAndRedirect } from '$lib/utils/auth';
	import Droplet from '@lucide/svelte/icons/droplet';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Moon from '@lucide/svelte/icons/moon';
	import Settings from '@lucide/svelte/icons/settings';
	import Sun from '@lucide/svelte/icons/sun';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	const onSettings = $derived(page.url.pathname.startsWith('/settings'));
</script>

<Toaster />

<div class="min-h-svh">
	<header class="bg-background sticky top-0 z-20 border-b">
		<div class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-5 py-3">
			<a href="/" class="flex items-center gap-2 text-lg font-semibold tracking-tight">
				<Droplet class="text-primary size-7" fill="currentColor" strokeWidth={0} />
				dropla
			</a>

			<nav class="flex items-center gap-1">
				<Button
					variant="ghost"
					size="icon"
					aria-label="Toggle dark mode"
					title="Toggle dark mode"
					onclick={() => theme.toggle()}
				>
					{#if theme.resolved === 'dark'}<Sun class="size-4" />{:else}<Moon class="size-4" />{/if}
				</Button>

				<Button
					variant={onSettings ? 'secondary' : 'ghost'}
					size="icon"
					href="/settings"
					aria-label="Settings"
					title="Settings — {data.user.username}"
				>
					<Settings class="size-4" />
				</Button>

				<Button
					variant="ghost"
					size="icon"
					aria-label="Sign out"
					title="Sign out"
					onclick={logoutAndRedirect}
				>
					<LogOut class="size-4" />
				</Button>
			</nav>
		</div>
	</header>

	<main class="mx-auto max-w-5xl px-5 py-8">
		{@render children()}
	</main>
</div>
