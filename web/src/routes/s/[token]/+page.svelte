<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Progress } from '$lib/components/ui/progress';
	import { Spinner } from '$lib/components/ui/spinner';
	import { Toaster } from '$lib/components/ui/sonner';
	import { apiErrorMessage, getPublicShare, unlockShare, type PublicShareDto } from '$lib/sdk';
	import { theme } from '$lib/theme.svelte';
	import { UploadBatch } from '$lib/transfer/upload.svelte';
	import { formatBytes } from '$lib/utils/format';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import Download from '@lucide/svelte/icons/download';
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import { onMount } from 'svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let share = $state<PublicShareDto | null>(null);
	let loading = $state(true);
	let notFound = $state(false);

	let password = $state('');
	let unlocking = $state(false);
	let unlockError = $state('');

	let uploaderLabel = $state('');
	let batch = $state<UploadBatch | null>(null);
	let fileInput = $state<HTMLInputElement | null>(null);
	let dragging = $state(false);

	// Above this the archive triggers Zip64, which Windows' built-in extractor
	// cannot open — and a streamed zip cannot resume. Past it, per-file downloads
	// become coequal rather than subordinate to "download all".
	const ZIP64_THRESHOLD = 4 * 1024 ** 3;
	const bigArchive = $derived((share?.totalBytes ?? 0) >= ZIP64_THRESHOLD);

	onMount(load);

	async function load() {
		try {
			share = await getPublicShare(data.token);
		} catch {
			notFound = true;
		} finally {
			loading = false;
		}
	}

	async function unlock(event: SubmitEvent) {
		event.preventDefault();
		unlocking = true;
		unlockError = '';
		try {
			share = await unlockShare(data.token, { password });
			password = '';
		} catch (e) {
			unlockError = apiErrorMessage(e, 'That password did not work.');
		} finally {
			unlocking = false;
		}
	}

	async function upload(files: File[]) {
		if (files.length === 0) return;

		const running = new UploadBatch(files, {
			shareToken: data.token,
			uploaderLabel: uploaderLabel.trim()
		});
		batch = running;
		await running.run();
		// Deliberately not reloading the Share: an Upload Box must never become
		// readable, so the confirmation below is client-side only and is gone on
		// reload. Refresh only the accepting/full state.
		try {
			share = await getPublicShare(data.token);
		} catch {
			/* keep whatever we had */
		}
	}
</script>

<svelte:head>
	<title>{share?.title ?? 'dropla'}</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<Toaster />

<div class="min-h-svh">
	<div class="absolute right-4 top-4">
		<Button
			variant="ghost"
			size="icon"
			aria-label="Toggle dark mode"
			onclick={() => theme.toggle()}
		>
			{#if theme.resolved === 'dark'}<Sun class="size-4" />{:else}<Moon class="size-4" />{/if}
		</Button>
	</div>

	<div class="mx-auto max-w-2xl px-5 pb-40 pt-[8vh]">
		{#if loading}
			<p class="text-muted-foreground text-sm">Loading…</p>
		{:else if notFound}
			<h1 class="text-3xl font-semibold tracking-tight">Not found</h1>
			<p class="text-muted-foreground mt-2">This link does not point to anything.</p>
		{:else if share}
			{#if share.state === 'expired'}
				<h1 class="text-3xl font-semibold tracking-tight">This share has expired</h1>
				<p class="text-muted-foreground mt-2">Ask the sender for a new link.</p>
			{:else if share.state === 'locked'}
				<h1 class="text-3xl font-semibold tracking-tight [overflow-wrap:anywhere]">
					{share.title}
				</h1>
				<p class="text-muted-foreground mt-2">This share is password protected.</p>
				<form class="mt-6 grid max-w-sm gap-3" onsubmit={unlock}>
					<Input
						type="password"
						bind:value={password}
						placeholder="Password"
						autocomplete="off"
						class="text-base"
					/>
					{#if unlockError}<p class="text-destructive text-sm">{unlockError}</p>{/if}
					<Button type="submit" class="justify-self-start" disabled={unlocking || !password}>
						{#if unlocking}<Spinner class="mr-2 size-4" />{/if}Unlock
					</Button>
				</form>
			{:else if share.mode === 'upload_box'}
				<h1 class="text-3xl font-semibold tracking-tight [overflow-wrap:anywhere]">
					Send files to {share.title}
				</h1>
				<p class="text-muted-foreground mt-2">Only the owner can see what is in this box.</p>

				{#if share.state === 'full'}
					<Card.Root class="mt-7">
						<Card.Content class="py-8 text-center">
							<p class="font-medium">This box is no longer accepting files.</p>
							<p class="text-muted-foreground mt-1 text-sm">
								Let the owner know if you still need to send something.
							</p>
						</Card.Content>
					</Card.Root>
				{:else}
					<div
						role="region"
						aria-label="Drop files to upload"
						class="border-primary/40 hover:bg-primary/5 mt-7 rounded-xl border-2 border-dashed p-12 text-center transition-colors {dragging
							? 'bg-primary/10'
							: ''}"
						ondragover={(e) => {
							e.preventDefault();
							dragging = true;
						}}
						ondragleave={() => (dragging = false)}
						ondrop={(e) => {
							e.preventDefault();
							dragging = false;
							void upload(Array.from(e.dataTransfer?.files ?? []));
						}}
					>
						<p class="text-lg font-medium">Drop files here</p>
						<Button class="mt-4" size="lg" onclick={() => fileInput?.click()}>Choose files</Button>
						<input
							bind:this={fileInput}
							type="file"
							multiple
							class="hidden"
							onchange={(e) => {
								const input = e.currentTarget;
								void upload(Array.from(input.files ?? []));
								input.value = '';
							}}
						/>
					</div>

					<div class="mt-5 grid max-w-sm gap-2">
						<Label for="uploader">Your name (optional)</Label>
						<Input
							id="uploader"
							bind:value={uploaderLabel}
							placeholder="So the owner knows who sent it"
							maxlength={120}
						/>
					</div>
				{/if}

				{#if batch}
					<Card.Root class="mt-6">
						<Card.Header>
							<Card.Title class="text-base">{batch.finished ? 'Sent' : 'Sending…'}</Card.Title>
							<Card.Description>
								This list is only on your screen and disappears when you reload.
							</Card.Description>
						</Card.Header>
						<Card.Content class="grid gap-2">
							{#each batch.items as item (item.file.name + item.file.size)}
								<div class="grid gap-1">
									<div class="flex items-center justify-between gap-3 text-sm">
										<span class="flex min-w-0 items-center gap-2">
											{#if item.status === 'done'}
												<CircleCheck class="size-4 shrink-0 text-green-600" />
											{:else if item.status === 'error'}
												<CircleX class="text-destructive size-4 shrink-0" />
											{/if}
											<span class="truncate">{item.file.name}</span>
										</span>
										<span class="text-muted-foreground shrink-0 text-xs">
											{item.status === 'done'
												? formatBytes(item.file.size)
												: item.status === 'error'
													? 'not accepted'
													: `${item.progress}%`}
										</span>
									</div>
									{#if item.status === 'uploading'}<Progress
											value={item.progress}
											class="h-1"
										/>{/if}
									{#if item.error}<p class="text-destructive text-xs">{item.error}</p>{/if}
								</div>
							{/each}
						</Card.Content>
					</Card.Root>
				{/if}
			{:else}
				<h1 class="text-3xl font-semibold tracking-tight [overflow-wrap:anywhere]">
					{share.title}
				</h1>
				<p class="text-muted-foreground mt-2">
					{share.files.length} file{share.files.length === 1 ? '' : 's'} · {formatBytes(
						share.totalBytes
					)}
				</p>

				{#if bigArchive}
					<p class="mt-4 rounded-lg border px-3 py-2 text-sm">
						Large share. Downloading files one at a time is more reliable.
					</p>
				{/if}

				<Card.Root class="mt-6">
					<Card.Content class="p-0">
						{#each share.files as file (file.downloadPath)}
							<a
								href={file.downloadPath}
								class="hover:bg-muted/50 flex items-center justify-between gap-3 border-b px-5 py-4 last:border-b-0"
							>
								<div class="min-w-0">
									<p class="truncate font-medium">{file.name}</p>
									<p class="text-muted-foreground text-xs">{formatBytes(file.size)}</p>
								</div>
								<Download class="size-4 shrink-0" />
							</a>
						{/each}
					</Card.Content>
				</Card.Root>
			{/if}
		{/if}
	</div>

	<!-- Sticky primary action, but honest: it always states the archive size, and
	     above the Zip64 threshold the per-file list above is promoted instead. -->
	{#if share && share.state === 'open' && share.mode === 'send' && share.files.length > 1}
		<div class="bg-background fixed inset-x-0 bottom-0 border-t px-5 py-3">
			<div class="mx-auto flex max-w-2xl items-center justify-between gap-4">
				<div class="min-w-0">
					<p class="font-medium">Download all</p>
					<p class="text-muted-foreground text-xs">
						{formatBytes(share.totalBytes)} zip · cannot resume if it fails
					</p>
				</div>
				<Button size="lg" variant={bigArchive ? 'outline' : 'default'} href={share.zipPath}>
					Get everything
				</Button>
			</div>
		</div>
	{/if}
</div>
