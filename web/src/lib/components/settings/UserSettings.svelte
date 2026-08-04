<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Spinner } from '$lib/components/ui/spinner';
	import * as Switch from '$lib/components/ui/switch';
	import { auth } from '$lib/managers/auth.svelte';
	import {
		apiErrorMessage,
		createUser,
		deleteUser,
		getUsers,
		resetUserPassword,
		setUserQuota,
		type UserDto
	} from '$lib/sdk';
	import { formatBytes, parseBytes } from '$lib/utils/format';
	import HardDrive from '@lucide/svelte/icons/hard-drive';
	import KeyRound from '@lucide/svelte/icons/key-round';
	import Plus from '@lucide/svelte/icons/plus';
	import Settings from '@lucide/svelte/icons/settings';
	import Shield from '@lucide/svelte/icons/shield';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';

	let users = $state<UserDto[]>([]);
	let loading = $state(true);
	let error = $state('');

	let addOpen = $state(false);
	let username = $state('');
	let password = $state('');
	let isAdmin = $state(false);
	let saving = $state(false);

	let pwOpen = $state(false);
	let pwTarget = $state<UserDto | null>(null);
	let newPassword = $state('');

	let quotaOpen = $state(false);
	let quotaTarget = $state<UserDto | null>(null);
	let quotaInput = $state('');

	onMount(refresh);

	async function refresh() {
		try {
			users = await getUsers();
			error = '';
		} catch (e) {
			error = apiErrorMessage(e);
		} finally {
			loading = false;
		}
	}

	async function add() {
		saving = true;
		try {
			await createUser({ username: username.trim(), password, isAdmin });
			await refresh();
			addOpen = false;
			username = '';
			password = '';
			isAdmin = false;
			toast.success('Account created');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			saving = false;
		}
	}

	async function remove(user: UserDto) {
		const ok = confirm(
			`Delete ${user.username}?\n\nThis also deletes every Share they own, the files inside them, ` +
				`and any links they handed out. It cannot be undone.`
		);
		if (!ok) return;

		try {
			await deleteUser(user.id);
			await refresh();
			toast.success(`Deleted ${user.username}`);
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}

	async function submitPassword() {
		if (!pwTarget) return;
		saving = true;
		try {
			await resetUserPassword(pwTarget.id, { newPassword });
			pwOpen = false;
			newPassword = '';
			toast.success('Password reset');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			saving = false;
		}
	}

	async function submitQuota() {
		if (!quotaTarget) return;
		saving = true;
		try {
			const trimmed = quotaInput.trim();
			const bytes = trimmed === '' ? null : parseBytes(trimmed);
			if (trimmed !== '' && bytes === null) {
				toast.error('Could not read that size. Try "50GB", "500 MB" or 0 for unlimited.');
				return;
			}
			await setUserQuota(quotaTarget.id, { quotaBytes: bytes });
			await refresh();
			quotaOpen = false;
			toast.success('Quota updated');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			saving = false;
		}
	}

	function quotaLabel(user: UserDto) {
		if (user.quotaBytes === undefined || user.quotaBytes === null) return 'server default';
		if (user.quotaBytes === 0) return 'unlimited';
		return formatBytes(user.quotaBytes);
	}
</script>

<div class="mb-4 flex items-end justify-between gap-4">
	<p class="text-muted-foreground text-sm">Accounts that can sign in to this instance.</p>
	<Button size="icon" variant="outline" aria-label="Add user" onclick={() => (addOpen = true)}>
		<Plus class="size-4" />
	</Button>
</div>

{#if loading}
	<p class="text-muted-foreground text-sm">Loading…</p>
{:else if error}
	<p class="text-destructive text-sm">{error}</p>
{:else}
	<div class="grid gap-3">
		{#each users as user (user.id)}
			<Card.Root>
				<Card.Content class="flex flex-wrap items-center justify-between gap-3">
					<div class="min-w-0">
						<div class="flex items-center gap-2">
							<p class="truncate font-medium">{user.username}</p>
							{#if user.isAdmin}
								<Badge variant="secondary" class="gap-1">
									<Shield class="size-3" /> Admin
								</Badge>
							{/if}
							{#if user.id === auth.user?.id}
								<Badge variant="outline">You</Badge>
							{/if}
						</div>
						<p class="text-muted-foreground mt-0.5 text-xs">Quota: {quotaLabel(user)}</p>
					</div>

					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Button {...props} variant="ghost" size="icon" aria-label="Manage {user.username}">
									<Settings class="size-4" />
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<!-- The content is width-clamped to the trigger by default, which for an
						     icon button is unusably narrow. -->
						<DropdownMenu.Content align="end" class="w-44">
							<DropdownMenu.Item
								onSelect={() => {
									quotaTarget = user;
									quotaInput =
										user.quotaBytes === undefined || user.quotaBytes === null
											? ''
											: String(user.quotaBytes);
									quotaOpen = true;
								}}
							>
								<HardDrive /> Storage quota
							</DropdownMenu.Item>
							<DropdownMenu.Item
								onSelect={() => {
									pwTarget = user;
									newPassword = '';
									pwOpen = true;
								}}
							>
								<KeyRound /> Reset password
							</DropdownMenu.Item>
							{#if user.id !== auth.user?.id}
								<DropdownMenu.Separator />
								<DropdownMenu.Item variant="destructive" onSelect={() => remove(user)}>
									<Trash2 /> Delete account
								</DropdownMenu.Item>
							{/if}
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
{/if}

<Dialog.Root bind:open={addOpen}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Add user</Dialog.Title>
			<Dialog.Description>Create an account.</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-4">
			<div class="grid gap-2">
				<Label for="new-username">Username</Label>
				<Input id="new-username" bind:value={username} autocomplete="off" />
			</div>
			<div class="grid gap-2">
				<Label for="new-password">Password</Label>
				<Input
					id="new-password"
					type="password"
					bind:value={password}
					autocomplete="new-password"
					placeholder="At least 8 characters"
				/>
			</div>
			<div class="flex items-center gap-3">
				<Switch.Root id="new-admin" bind:checked={isAdmin} />
				<div>
					<Label for="new-admin">Administrator</Label>
					<p class="text-muted-foreground text-xs">Can manage accounts and quotas.</p>
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<Dialog.Close type="button" class={buttonVariants({ variant: 'outline' })}
				>Cancel</Dialog.Close
			>
			<Button onclick={add} disabled={saving || !username.trim() || password.length < 8}>
				{#if saving}<Spinner class="mr-2 size-4" />{/if}Add user
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={pwOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Reset password</Dialog.Title>
			<Dialog.Description>Signs {pwTarget?.username} out everywhere.</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-2">
			<Label for="reset-password">New password</Label>
			<Input
				id="reset-password"
				type="password"
				bind:value={newPassword}
				autocomplete="new-password"
				placeholder="At least 8 characters"
			/>
		</div>
		<Dialog.Footer>
			<Dialog.Close type="button" class={buttonVariants({ variant: 'outline' })}
				>Cancel</Dialog.Close
			>
			<Button onclick={submitPassword} disabled={saving || newPassword.length < 8}>
				{#if saving}<Spinner class="mr-2 size-4" />{/if}Reset
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={quotaOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Storage quota</Dialog.Title>
			<Dialog.Description>Total storage for {quotaTarget?.username}.</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-2">
			<Label for="quota">Quota</Label>
			<Input id="quota" bind:value={quotaInput} placeholder="50GB" />
			<p class="text-muted-foreground text-xs">Blank for the default, 0 for unlimited.</p>
		</div>
		<Dialog.Footer>
			<Dialog.Close type="button" class={buttonVariants({ variant: 'outline' })}
				>Cancel</Dialog.Close
			>
			<Button onclick={submitQuota} disabled={saving}>
				{#if saving}<Spinner class="mr-2 size-4" />{/if}Save
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
