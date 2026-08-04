<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { apiErrorMessage, type ShareDto } from '$lib/sdk';
	import { formatBytes, formatExpiry } from '$lib/utils/format';
	import Check from '@lucide/svelte/icons/check';
	import Copy from '@lucide/svelte/icons/copy';
	import Lock from '@lucide/svelte/icons/lock';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import { toast } from 'svelte-sonner';

	let { share, onChanged }: { share: ShareDto; onChanged?: () => void } = $props();

	let copied = $state(false);

	async function copyLink() {
		const url = share.publicUrl || new URL(share.path, location.origin).toString();
		try {
			await navigator.clipboard.writeText(url);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch (e) {
			toast.error(apiErrorMessage(e, 'Could not copy to clipboard'));
		}
		onChanged?.();
	}
</script>

<!--
	The whole row opens the share. A button cannot sit inside an anchor, so the
	link is a stretched overlay underneath the copy control rather than a wrapper.
-->
<Card.Root
	class="hover:ring-primary/70 focus-within:ring-primary relative transition-shadow hover:ring-2 focus-within:ring-2"
>
	<Card.Content class="flex flex-wrap items-center justify-between gap-3">
		<div class="min-w-0 flex-1">
			<div class="flex min-w-0 flex-wrap items-center gap-2">
				<p class="min-w-0 truncate font-medium">{share.title}</p>
				{#if share.hasPassword}
					<Badge variant="outline" class="gap-1"><Lock class="size-3" /> Password</Badge>
				{/if}
				{#if share.expired}
					<Badge variant="destructive">Expired</Badge>
				{/if}
			</div>

			<p class="text-muted-foreground mt-0.5 text-xs">
				{share.files.length} file{share.files.length === 1 ? '' : 's'} ·
				{formatBytes(share.totalBytes)} · expires {formatExpiry(share.expiresAt)}
				{#if share.maxDownloads}
					· {share.downloads}/{share.maxDownloads} downloads
				{:else if share.mode === 'send'}
					· {share.downloads} downloads
				{/if}
			</p>

			{#if share.notifyLastError}
				<p class="text-destructive mt-1.5 flex items-start gap-1.5 text-xs">
					<TriangleAlert class="mt-0.5 size-3.5 shrink-0" />
					<span>Notification failed. {share.notifyLastError}</span>
				</p>
			{/if}
		</div>

		<Button variant="ghost" size="sm" class="relative z-10" onclick={copyLink}>
			{#if copied}
				<Check class="mr-1.5 size-3.5" /> Copied
			{:else}
				<Copy class="mr-1.5 size-3.5" /> Copy link
			{/if}
		</Button>
	</Card.Content>

	<a href="/shares/{share.id}" class="absolute inset-0 z-0 rounded-xl">
		<span class="sr-only">Open {share.title}</span>
	</a>
</Card.Root>
