<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Spinner } from '$lib/components/ui/spinner';
	import { auth } from '$lib/managers/auth.svelte';
	import {
		apiErrorMessage,
		changePassword,
		createAccessToken,
		deleteAccessToken,
		getAccessTokens,
		type AccessTokenDto
	} from '$lib/sdk';
	import { theme, type Theme } from '$lib/theme.svelte';
	import { formatDate } from '$lib/utils/format';
	import Copy from '@lucide/svelte/icons/copy';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let changing = $state(false);

	let tokens = $state<AccessTokenDto[]>([]);
	let tokenName = $state('');
	let creating = $state(false);
	let createOpen = $state(false);

	// The secret is readable exactly once, so it is held in memory only until the
	// dialog closes. Nothing re-displays it afterwards.
	let issued = $state<{ token: string; command: string } | null>(null);

	const themes: { value: Theme; label: string }[] = [
		{ value: 'light', label: 'Light' },
		{ value: 'dark', label: 'Dark' },
		{ value: 'system', label: 'System' }
	];

	onMount(loadTokens);

	async function loadTokens() {
		try {
			tokens = await getAccessTokens();
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}

	async function submitPassword(event: SubmitEvent) {
		event.preventDefault();
		if (newPassword !== confirmPassword) {
			toast.error('The new passwords do not match');
			return;
		}

		changing = true;
		try {
			await changePassword({ currentPassword, newPassword });
			toast.success('Password changed. Sign in again.');
			auth.user = null;
			await goto('/login');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			changing = false;
		}
	}

	async function issueToken() {
		creating = true;
		try {
			const created = await createAccessToken({ name: tokenName.trim() });
			issued = { token: created.token, command: created.command };
			tokenName = '';
			createOpen = false;
			await loadTokens();
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			creating = false;
		}
	}

	async function revoke(token: AccessTokenDto) {
		if (!confirm(`Revoke "${token.name}"? Scripts using it stop working immediately.`)) return;
		try {
			await deleteAccessToken(token.id);
			await loadTokens();
			toast.success('Token revoked');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}

	async function copy(text: string, what: string) {
		try {
			await navigator.clipboard.writeText(text);
			toast.success(`${what} copied`);
		} catch {
			toast.error('Could not copy to clipboard');
		}
	}
</script>

<div class="grid gap-6">
	<Card.Root>
		<Card.Header>
			<Card.Title class="text-base">Appearance</Card.Title>
		</Card.Header>
		<Card.Content>
			<div class="flex gap-2">
				{#each themes as option (option.value)}
					<Button
						variant={theme.preference === option.value ? 'default' : 'outline'}
						size="sm"
						onclick={() => theme.set(option.value)}
					>
						{option.label}
					</Button>
				{/each}
			</div>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title class="text-base">Change password</Card.Title>
			<Card.Description>You will need to sign in again.</Card.Description>
		</Card.Header>
		<Card.Content>
			<form class="grid max-w-sm gap-4" onsubmit={submitPassword}>
				<div class="grid gap-2">
					<Label for="current">Current password</Label>
					<Input
						id="current"
						type="password"
						bind:value={currentPassword}
						autocomplete="current-password"
						required
					/>
				</div>
				<div class="grid gap-2">
					<Label for="new">New password</Label>
					<Input
						id="new"
						type="password"
						bind:value={newPassword}
						autocomplete="new-password"
						placeholder="At least 8 characters"
						required
					/>
				</div>
				<div class="grid gap-2">
					<Label for="confirm">Confirm new password</Label>
					<Input
						id="confirm"
						type="password"
						bind:value={confirmPassword}
						autocomplete="new-password"
						required
					/>
				</div>
				<Button
					type="submit"
					class="justify-self-start"
					disabled={changing || newPassword.length < 8 || !currentPassword}
				>
					{#if changing}<Spinner class="mr-2 size-4" />{/if}Change password
				</Button>
			</form>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title class="text-base">CLI tokens</Card.Title>
			<Card.Description>For uploading from a terminal.</Card.Description>
			<Card.Action>
				<Button
					size="icon"
					variant="outline"
					aria-label="New token"
					onclick={() => (createOpen = true)}
				>
					<Plus class="size-4" />
				</Button>
			</Card.Action>
		</Card.Header>
		<Card.Content>
			{#if tokens.length === 0}
				<p class="text-muted-foreground text-sm">No tokens yet.</p>
			{:else}
				<div class="grid gap-2">
					{#each tokens as token (token.id)}
						<div class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
							<div class="min-w-0">
								<p class="truncate text-sm font-medium">{token.name}</p>
								<p class="text-muted-foreground text-xs">
									Created {formatDate(token.createdAt)} · Last used {token.lastUsedAt
										? formatDate(token.lastUsedAt)
										: 'never'}
								</p>
							</div>
							<Button
								variant="ghost"
								size="icon"
								aria-label="Revoke {token.name}"
								onclick={() => revoke(token)}
							>
								<Trash2 class="text-destructive size-4" />
							</Button>
						</div>
					{/each}
				</div>
			{/if}
		</Card.Content>
	</Card.Root>
</div>

<Dialog.Root bind:open={createOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>New CLI token</Dialog.Title>
			<Dialog.Description>Name it after where it will be used.</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-2">
			<Label for="token-name">Name</Label>
			<Input id="token-name" bind:value={tokenName} placeholder="laptop" />
		</div>
		<Dialog.Footer>
			<Dialog.Close type="button" class={buttonVariants({ variant: 'outline' })}
				>Cancel</Dialog.Close
			>
			<Button onclick={issueToken} disabled={creating || !tokenName.trim()}>
				{#if creating}<Spinner class="mr-2 size-4" />{/if}Create
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root
	open={issued !== null}
	onOpenChange={(open) => {
		if (!open) issued = null;
	}}
>
	<Dialog.Content class="sm:max-w-xl">
		<Dialog.Header>
			<Dialog.Title>Token created</Dialog.Title>
			<Dialog.Description>Copy this now. It will not be shown again.</Dialog.Description>
		</Dialog.Header>
		{#if issued}
			<div class="grid gap-3">
				<div class="grid gap-1.5">
					<Label>Ready-to-run command</Label>
					<pre class="bg-muted overflow-x-auto rounded-lg border p-3 text-xs">{issued.command}</pre>
					<Button
						variant="outline"
						size="sm"
						class="justify-self-start"
						onclick={() => copy(issued!.command, 'Command')}
					>
						<Copy class="mr-2 size-3.5" /> Copy command
					</Button>
				</div>
				<div class="grid gap-1.5">
					<Label>Token</Label>
					<code class="bg-muted overflow-x-auto rounded-lg border p-3 text-xs">{issued.token}</code>
					<Button
						variant="outline"
						size="sm"
						class="justify-self-start"
						onclick={() => copy(issued!.token, 'Token')}
					>
						<Copy class="mr-2 size-3.5" /> Copy token
					</Button>
				</div>
			</div>
		{/if}
		<Dialog.Footer>
			<Dialog.Close class={buttonVariants()}>Done</Dialog.Close>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
