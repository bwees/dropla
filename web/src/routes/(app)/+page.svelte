<script lang="ts">
	import { goto } from '$app/navigation';
	import CreateBoxDialog from '$lib/components/CreateBoxDialog.svelte';
	import ShareRow from '$lib/components/ShareRow.svelte';
	import UploadProgress from '$lib/components/UploadProgress.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Progress } from '$lib/components/ui/progress';
	import * as Tabs from '$lib/components/ui/tabs';
	import { apiErrorMessage, createShare, getShares, getUsage, type ShareDto } from '$lib/sdk';
	import { UploadBatch } from '$lib/transfer/upload.svelte';
	import { formatBytes } from '$lib/utils/format';
	import { shareToken } from '$lib/utils/share';
	import Inbox from '@lucide/svelte/icons/inbox';
	import Plus from '@lucide/svelte/icons/plus';
	import Upload from '@lucide/svelte/icons/upload';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';

	let shares = $state<ShareDto[]>([]);
	let usage = $state<{ usedBytes: number; quotaBytes: number } | null>(null);
	let loading = $state(true);

	let tab = $state('send');
	let dragging = $state(false);
	let batch = $state<UploadBatch | null>(null);
	let boxOpen = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	const sendShares = $derived(shares.filter((s) => s.mode === 'send'));
	const uploadBoxes = $derived(shares.filter((s) => s.mode === 'upload_box'));

	onMount(refresh);

	async function refresh() {
		try {
			[shares, usage] = await Promise.all([getShares(), getUsage()]);
		} catch (e) {
			toast.error(apiErrorMessage(e));
		} finally {
			loading = false;
		}
	}

	// The share is created before the first byte moves, which is also what tus
	// requires: an upload has to have a target.
	async function send(files: File[]) {
		if (files.length === 0) return;
		// A drop can land while the Upload Boxes tab is showing; the result belongs
		// under Shares, so move there rather than uploading somewhere invisible.
		tab = 'send';

		const title = files.length === 1 ? files[0].name : `${files.length} files`;

		let share: ShareDto;
		let token: string;
		try {
			share = await createShare({ title, mode: 'send' });
			token = shareToken(share);
		} catch (e) {
			toast.error(apiErrorMessage(e, 'Could not create the share.'));
			return;
		}

		const running = new UploadBatch(files, { shareToken: token });
		batch = running;

		await running.run();
		await refresh();

		const failed = running.items.filter((i) => i.status === 'error');
		if (failed.length === 0) {
			toast.success(files.length === 1 ? 'File uploaded' : `${files.length} files uploaded`);
			batch = null;
			await goto(`/shares/${share.id}`);
		} else {
			toast.error(`${failed.length} of ${files.length} failed`);
		}
	}

	async function createBox(payload: Parameters<typeof createShare>[0]) {
		try {
			const share = await createShare(payload);
			boxOpen = false;
			toast.success('Upload Box created');
			await goto(`/shares/${share.id}`);
		} catch (e) {
			toast.error(apiErrorMessage(e));
		}
	}
</script>

<svelte:head><title>dropla</title></svelte:head>

<svelte:window
	ondragover={(e) => {
		e.preventDefault();
		dragging = true;
	}}
	ondragleave={() => (dragging = false)}
	ondrop={(e) => {
		e.preventDefault();
		dragging = false;
		void send(Array.from(e.dataTransfer?.files ?? []));
	}}
/>

<Tabs.Root bind:value={tab}>
	<div class="mb-6 flex flex-wrap items-center justify-between gap-4">
		<Tabs.List>
			<Tabs.Trigger value="send">
				<Upload /> Shares
				{#if sendShares.length}
					<span class="text-muted-foreground">{sendShares.length}</span>
				{/if}
			</Tabs.Trigger>
			<Tabs.Trigger value="boxes">
				<Inbox /> Upload Boxes
				{#if uploadBoxes.length}
					<span class="text-muted-foreground">{uploadBoxes.length}</span>
				{/if}
			</Tabs.Trigger>
		</Tabs.List>

		{#if usage}
			<div class="w-48 text-right">
				<p class="text-sm">
					<span class="font-medium">{formatBytes(usage.usedBytes)}</span>
					<span class="text-muted-foreground">
						{usage.quotaBytes > 0 ? `of ${formatBytes(usage.quotaBytes)}` : 'used'}
					</span>
				</p>
				{#if usage.quotaBytes > 0}
					<Progress value={(usage.usedBytes / usage.quotaBytes) * 100} class="mt-1.5 h-1.5" />
				{/if}
			</div>
		{/if}
	</div>

	<Tabs.Content value="send">
		<div
			role="region"
			aria-label="Drop files to send"
			class="border-primary/40 hover:bg-primary/5 rounded-xl border-2 border-dashed p-12 text-center transition-colors {dragging
				? 'bg-primary/10'
				: ''}"
		>
			<p class="font-medium">Drag files anywhere on this page</p>
			<Button class="mt-4" onclick={() => fileInput?.click()}>
				<Upload class="mr-2 size-4" /> Choose files to send
			</Button>
			<input
				bind:this={fileInput}
				type="file"
				multiple
				class="hidden"
				onchange={(e) => {
					const input = e.currentTarget;
					void send(Array.from(input.files ?? []));
					input.value = '';
				}}
			/>
		</div>

		{#if batch}
			<UploadProgress {batch} onDismiss={() => (batch = null)} />
		{/if}

		<div class="mt-8 grid gap-2">
			{#if loading}
				<p class="text-muted-foreground text-sm">Loading…</p>
			{:else if sendShares.length === 0}
				<p class="text-muted-foreground text-sm">Nothing shared yet.</p>
			{:else}
				{#each sendShares as share (share.id)}
					<ShareRow {share} onChanged={refresh} />
				{/each}
			{/if}
		</div>
	</Tabs.Content>

	<Tabs.Content value="boxes">
		<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
			<p class="text-muted-foreground text-sm">
				Let someone send you files without seeing what is inside.
			</p>
			<Button onclick={() => (boxOpen = true)}>
				<Plus class="mr-2 size-4" /> New Upload Box
			</Button>
		</div>

		<div class="grid gap-2">
			{#if loading}
				<p class="text-muted-foreground text-sm">Loading…</p>
			{:else if uploadBoxes.length === 0}
				<p class="text-muted-foreground text-sm">No Upload Boxes yet.</p>
			{:else}
				{#each uploadBoxes as share (share.id)}
					<ShareRow {share} onChanged={refresh} />
				{/each}
			{/if}
		</div>
	</Tabs.Content>
</Tabs.Root>

<CreateBoxDialog bind:open={boxOpen} onCreate={createBox} />
