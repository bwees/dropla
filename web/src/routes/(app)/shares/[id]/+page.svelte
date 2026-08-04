<script lang="ts">
	import { goto } from '$app/navigation';
	import HeaderLink from '$lib/components/HeaderLink.svelte';
	import PasswordField from '$lib/components/PasswordField.svelte';
	import SizeField from '$lib/components/SizeField.svelte';
	import UploadProgress from '$lib/components/UploadProgress.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';
	import { Separator } from '$lib/components/ui/separator';
	import { Spinner } from '$lib/components/ui/spinner';
	import * as Switch from '$lib/components/ui/switch';
	import {
		apiErrorMessage,
		deleteShare,
		deleteShareFile,
		getShare,
		rotateShareLink,
		updateShare,
		type ShareDto,
		type UpdateShareDto
	} from '$lib/sdk';
	import { UploadBatch } from '$lib/transfer/upload.svelte';
	import { formatBytes, formatDate, formatExpiry } from '$lib/utils/format';
	import { shareToken } from '$lib/utils/share';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Download from '@lucide/svelte/icons/download';
	import Inbox from '@lucide/svelte/icons/inbox';
	import Lock from '@lucide/svelte/icons/lock';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let share = $state<ShareDto | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let batch = $state<UploadBatch | null>(null);
	let fileInput = $state<HTMLInputElement | null>(null);
	let dropping = $state(false);

	let title = $state('');
	let expiry = $state('0');
	let currentExpiryDays = $state(0);
	let maxDownloads = $state('');
	let maxFiles = $state('');
	let maxBytes = $state(0);
	let passwordOn = $state(false);
	let password = $state('');
	let notifyUrl = $state('');
	let notifyEnabled = $state(false);

	const isBox = $derived(share?.mode === 'upload_box');

	const EXPIRY_CHOICES = [
		{ value: '0', label: 'Never expires' },
		{ value: '1', label: 'In 1 day' },
		{ value: '7', label: 'In 7 days' },
		{ value: '30', label: 'In 30 days' },
		{ value: '90', label: 'In 90 days' },
		{ value: '365', label: 'In a year' }
	];

	// The select shows what the share is actually set to. An expiry that does not
	// land on one of the presets keeps its own entry so opening the page and saving
	// cannot silently move the date.
	const expiryChoices = $derived.by(() => {
		if (!currentExpiryDays || EXPIRY_CHOICES.some((c) => c.value === String(currentExpiryDays))) {
			return EXPIRY_CHOICES;
		}
		return [
			{ value: String(currentExpiryDays), label: `In ${currentExpiryDays} days (current)` },
			...EXPIRY_CHOICES
		];
	});
	const expiryLabel = $derived(
		expiryChoices.find((c) => c.value === expiry)?.label ?? 'Never expires'
	);

	onMount(load);

	async function load() {
		try {
			const fetched = await getShare(data.id);
			share = fetched;
			title = fetched.title;
			maxDownloads = fetched.maxDownloads ? String(fetched.maxDownloads) : '';
			maxFiles = fetched.maxFiles ? String(fetched.maxFiles) : '';
			maxBytes = fetched.maxBytes ?? 0;
			notifyUrl = fetched.notifyUrl;
			notifyEnabled = fetched.notifyEnabled;
			passwordOn = fetched.hasPassword;
			// The password comes back in the clear, so the field shows what is set
			// rather than asking the owner to remember it.
			password = fetched.password;

			currentExpiryDays = fetched.expiresAt
				? Math.max(
						1,
						Math.round((new Date(fetched.expiresAt).getTime() - Date.now()) / (24 * 60 * 60 * 1000))
					)
				: 0;
			expiry = String(currentExpiryDays);
		} catch (e) {
			toast.error(apiErrorMessage(e));
			await goto('/');
		} finally {
			loading = false;
		}
	}

	async function save() {
		if (!share) return;

		const payload: UpdateShareDto = {};
		if (title.trim() && title.trim() !== share.title) payload.title = title.trim();
		if (Number(expiry) !== currentExpiryDays) payload.expiresInDays = Number(expiry);

		const wantedDownloads = maxDownloads.trim() ? Number(maxDownloads) : 0;
		if (wantedDownloads !== (share.maxDownloads ?? 0)) payload.maxDownloads = wantedDownloads;

		if (isBox) {
			const wantedFiles = maxFiles.trim() ? Number(maxFiles) : 0;
			if (wantedFiles !== (share.maxFiles ?? 0)) payload.maxFiles = wantedFiles;
			if (maxBytes !== (share.maxBytes ?? 0)) payload.maxBytes = maxBytes;
			if (notifyUrl.trim() !== share.notifyUrl) payload.notifyUrl = notifyUrl.trim();
			if (notifyEnabled !== share.notifyEnabled) payload.notifyEnabled = notifyEnabled;
		}

		if (!passwordOn && share.hasPassword) payload.password = '';
		else if (passwordOn && password && password !== share.password) payload.password = password;

		if (Object.keys(payload).length === 0) {
			toast.info('Nothing to save');
			return;
		}

		saving = true;
		try {
			await updateShare(share.id, payload);
			toast.success('Saved');
			await load();
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			saving = false;
		}
	}

	async function rotate() {
		if (!share) return;
		const ok = confirm(
			'Issue a new link?\n\nEvery copy of the current URL stops working immediately. The files, ' +
				'settings and download count are kept.'
		);
		if (!ok) return;

		try {
			share = await rotateShareLink(share.id);
			toast.success('New link issued');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}

	async function removeFile(fileId: string, name: string) {
		if (!share || !confirm(`Delete ${name}? This cannot be undone.`)) return;
		try {
			await deleteShareFile(share.id, fileId);
			toast.success('File deleted');
			await load();
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}

	async function removeShare() {
		if (!share) return;
		if (!confirm(`Delete "${share.title}" and everything in it? This cannot be undone.`)) return;
		try {
			await deleteShare(share.id);
			toast.success('Share deleted');
			await goto('/');
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}

	async function addFiles(files: File[]) {
		if (!share || files.length === 0) return;

		let token: string;
		try {
			token = shareToken(share);
		} catch (e) {
			toast.error(apiErrorMessage(e));
			return;
		}

		const running = new UploadBatch(files, { shareToken: token });
		batch = running;
		await running.run();
		await load();

		const failed = running.items.filter((i) => i.status === 'error');
		if (failed.length === 0) {
			toast.success(files.length === 1 ? 'File uploaded' : `${files.length} files uploaded`);
			batch = null;
		} else {
			toast.error(`${failed.length} of ${files.length} failed`);
		}
	}
</script>

<svelte:head><title>{share?.title ?? 'Share'} — dropla</title></svelte:head>

{#if loading}
	<p class="text-muted-foreground text-sm">Loading…</p>
{:else if share}
	<a
		href="/"
		class="text-muted-foreground hover:text-foreground mb-4 inline-flex items-center gap-1.5 text-sm"
	>
		<ArrowLeft class="size-3.5" /> All shares
	</a>

	<header class="mb-6 grid items-start gap-x-6 gap-y-3 sm:grid-cols-[minmax(0,1fr)_auto]">
		<div class="min-w-0">
			<h1 class="text-2xl font-semibold tracking-tight [overflow-wrap:anywhere]">
				{share.title}
			</h1>

			<div class="mt-2 flex flex-wrap items-center gap-1.5">
				{#if isBox}
					<Badge variant="secondary" class="gap-1"><Inbox class="size-3" /> Upload Box</Badge>
				{/if}
				<Badge variant="secondary">
					{share.files.length} file{share.files.length === 1 ? '' : 's'}
				</Badge>
				<Badge variant="secondary">{formatBytes(share.totalBytes)}</Badge>
				<Badge variant="secondary">
					{share.expiresAt ? `Expires ${formatExpiry(share.expiresAt)}` : 'Never expires'}
				</Badge>
				{#if !isBox}
					<Badge variant="secondary">
						{share.downloads}{share.maxDownloads ? ` / ${share.maxDownloads}` : ''} download{share.downloads ===
						1
							? ''
							: 's'}
					</Badge>
				{/if}
				{#if share.hasPassword}
					<Badge variant="outline" class="gap-1"><Lock class="size-3" /> Password</Badge>
				{/if}
				{#if share.expired}
					<Badge variant="destructive">Expired</Badge>
				{/if}
			</div>
		</div>

		<div class="w-full min-w-0 sm:w-72 sm:justify-self-end">
			<HeaderLink path={share.path} publicUrl={share.publicUrl} onRotate={rotate} />
		</div>
	</header>

	{#if share.notifyLastError}
		<div
			class="text-destructive border-destructive/40 mb-6 flex items-start gap-2 rounded-lg border px-3 py-2 text-sm"
		>
			<TriangleAlert class="mt-0.5 size-4 shrink-0" />
			<div>
				<p class="font-medium">Notifications are failing</p>
				<p class="text-xs">
					{share.notifyLastError} — last tried {formatDate(share.notifyLastErrorAt)}
				</p>
			</div>
		</div>
	{/if}

	<div class="grid gap-6">
		<Card.Root>
			<Card.Header>
				<Card.Title class="text-base">{isBox ? 'Received files' : 'Files'}</Card.Title>
				{#if isBox}
					<Card.Description>Only you can see these.</Card.Description>
				{/if}
				<Card.Action>
					<div class="flex gap-2">
						{#if share.files.length > 1}
							<Button variant="outline" size="sm" href="/api/shares/{share.id}/all.zip">
								<Download class="mr-1.5 size-3.5" /> Download all
							</Button>
						{/if}
					</div>
				</Card.Action>
			</Card.Header>
			<Card.Content>
				{#if share.files.length === 0}
					<p class="text-muted-foreground py-2 text-sm">
						{isBox ? 'Nothing has arrived yet.' : 'No files yet.'}
					</p>
				{:else}
					<ul class="divide-y">
						{#each share.files as file (file.id)}
							<li class="flex items-center justify-between gap-3 py-2.5 first:pt-0 last:pb-0">
								<div class="min-w-0">
									<p class="truncate text-sm font-medium">{file.name}</p>
									<p class="text-muted-foreground text-xs">
										{formatBytes(file.size)}
										{#if file.uploaderLabel}· from “{file.uploaderLabel}”{/if}
										· {formatDate(file.uploadedAt)}
									</p>
								</div>
								<div class="flex shrink-0 items-center">
									<Button
										variant="ghost"
										size="icon"
										href={file.downloadPath}
										aria-label="Download {file.name}"
									>
										<Download class="size-4" />
									</Button>
									<Button
										variant="ghost"
										size="icon"
										aria-label="Delete {file.name}"
										onclick={() => removeFile(file.id, file.name)}
									>
										<Trash2 class="text-destructive size-4" />
									</Button>
								</div>
							</li>
						{/each}
					</ul>
				{/if}

				{#if !isBox}
					<button
						type="button"
						class="border-primary/40 hover:bg-primary/5 mt-3 w-full rounded-lg border-2 border-dashed py-6 text-center transition-colors {dropping
							? 'bg-primary/10'
							: ''}"
						ondragover={(e) => {
							e.preventDefault();
							dropping = true;
						}}
						ondragleave={() => (dropping = false)}
						ondrop={(e) => {
							e.preventDefault();
							dropping = false;
							void addFiles(Array.from(e.dataTransfer?.files ?? []));
						}}
						onclick={() => fileInput?.click()}
					>
						<span class="text-muted-foreground text-sm"> Drop files here, or click to choose </span>
					</button>
				{/if}

				<input
					bind:this={fileInput}
					type="file"
					multiple
					class="hidden"
					onchange={(e) => {
						const input = e.currentTarget;
						void addFiles(Array.from(input.files ?? []));
						input.value = '';
					}}
				/>
			</Card.Content>
		</Card.Root>

		{#if batch}
			<UploadProgress {batch} onDismiss={() => (batch = null)} />
		{/if}

		<Card.Root>
			<Card.Header>
				<Card.Title class="text-base">Settings</Card.Title>
			</Card.Header>
			<Card.Content class="grid gap-5">
				<div class="grid gap-1.5">
					<Label for="title">Name</Label>
					<Input id="title" bind:value={title} />
				</div>

				<div class="grid gap-1.5">
					<Label for="expiry">Expiry</Label>
					<Select.Root type="single" bind:value={expiry}>
						<Select.Trigger id="expiry">{expiryLabel}</Select.Trigger>
						<Select.Content>
							{#each expiryChoices as choice (choice.value)}
								<Select.Item value={choice.value} label={choice.label}>{choice.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>

				{#if isBox}
					<div class="grid gap-1.5">
						<Label for="maxfiles">Maximum number of files</Label>
						<Input id="maxfiles" bind:value={maxFiles} inputmode="numeric" placeholder="No limit" />
					</div>

					<SizeField bind:bytes={maxBytes} id="maxbytes" label="Maximum total size" />
				{:else}
					<div class="grid gap-1.5">
						<Label for="cap">Download limit</Label>
						<Input id="cap" bind:value={maxDownloads} inputmode="numeric" placeholder="No limit" />
					</div>
				{/if}

				<Separator />

				<PasswordField bind:enabled={passwordOn} bind:value={password} />

				{#if isBox}
					<Separator />

					<div class="grid gap-3">
						<div class="flex items-start gap-3">
							<Switch.Root id="notify" bind:checked={notifyEnabled} />
							<Label for="notify">Notify me when files arrive</Label>
						</div>

						{#if notifyEnabled}
							<div class="grid gap-1.5 pl-11">
								<Input
									id="notify-url"
									bind:value={notifyUrl}
									placeholder="ntfy://ntfy.sh/my-topic"
								/>
							</div>
						{/if}
					</div>
				{/if}

				<div class="flex items-center gap-2">
					<Button onclick={save} disabled={saving}>
						{#if saving}<Spinner class="mr-2 size-4" />{/if}Save changes
					</Button>
					<Button variant="ghost" onclick={load} disabled={saving}>Reset</Button>
				</div>
			</Card.Content>
		</Card.Root>

		<Card.Root class="border-destructive/40">
			<Card.Header>
				<Card.Title class="text-base">Delete this share</Card.Title>
				<Card.Description>This cannot be undone.</Card.Description>
				<Card.Action>
					<Button variant="outline" size="sm" onclick={removeShare}>
						<Trash2 class="text-destructive mr-2 size-3.5" /> Delete
					</Button>
				</Card.Action>
			</Card.Header>
		</Card.Root>
	</div>
{/if}
