<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Progress } from '$lib/components/ui/progress';
	import type { UploadBatch } from '$lib/transfer/upload.svelte';
	import { formatBytes } from '$lib/utils/format';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';

	let { batch, onDismiss }: { batch: UploadBatch; onDismiss: () => void } = $props();
</script>

<Card.Root class="mt-5">
	<Card.Header>
		<Card.Title class="text-base">
			{batch.finished ? 'Upload finished' : 'Uploading…'}
		</Card.Title>
		{#if batch.allSettled}
			<Card.Action
				><Button variant="ghost" size="sm" onclick={onDismiss}>Dismiss</Button></Card.Action
			>
		{/if}
	</Card.Header>
	<Card.Content class="grid gap-3">
		{#if !batch.finished}
			<Progress value={batch.overallProgress} class="h-2" />
		{/if}

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
						{#if item.status === 'error'}
							failed
						{:else if item.status === 'done'}
							{formatBytes(item.file.size)}
						{:else}
							{item.progress}%
						{/if}
					</span>
				</div>
				{#if item.status === 'uploading'}
					<Progress value={item.progress} class="h-1" />
				{/if}
				{#if item.error}
					<p class="text-destructive text-xs">{item.error}</p>
				{/if}
			</div>
		{/each}
	</Card.Content>
</Card.Root>
