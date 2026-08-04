<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { auth } from '$lib/managers/auth.svelte';
	import { apiErrorMessage } from '$lib/sdk';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const setupMode = $derived(data.setupRequired);

	let username = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		loading = true;
		try {
			await (setupMode ? auth.setup(username, password) : auth.login(username, password));
			await goto('/');
		} catch (e) {
			error = apiErrorMessage(e);
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>{setupMode ? 'Set up dropla' : 'Sign in — dropla'}</title></svelte:head>

<div class="flex min-h-svh items-center justify-center p-4">
	<Card.Root class="w-full sm:max-w-md">
		<Card.Header>
			<Card.Title>{setupMode ? 'Create the first account' : 'Sign in'}</Card.Title>
			<Card.Description>
				{setupMode
					? 'No accounts exist yet. This one becomes the administrator.'
					: 'Enter your credentials to continue.'}
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<form class="grid gap-4" onsubmit={submit}>
				<div class="grid gap-2">
					<Label for="username">Username</Label>
					<Input id="username" bind:value={username} autocomplete="username" required />
				</div>
				<div class="grid gap-2">
					<Label for="password">Password</Label>
					<Input
						id="password"
						type="password"
						bind:value={password}
						autocomplete={setupMode ? 'new-password' : 'current-password'}
						required
					/>
					{#if setupMode}
						<p class="text-muted-foreground text-xs">At least 8 characters.</p>
					{/if}
				</div>
				{#if error}
					<p class="text-destructive text-sm">{error}</p>
				{/if}
				<Button type="submit" class="w-full" disabled={loading}>
					{loading ? 'Working…' : setupMode ? 'Create account' : 'Sign in'}
				</Button>
			</form>
		</Card.Content>
	</Card.Root>
</div>
