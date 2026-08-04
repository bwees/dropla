<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import Check from '@lucide/svelte/icons/check';
	import Copy from '@lucide/svelte/icons/copy';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import { toast } from 'svelte-sonner';

	// The private URL is composed here rather than server-side, because the browser
	// knows which origin the owner actually used.
	let { path, publicUrl, onRotate }: { path: string; publicUrl: string; onRotate?: () => void } =
		$props();

	const url = $derived(
		publicUrl ||
			(typeof location === 'undefined' ? path : new URL(path, location.origin).toString())
	);
	const display = $derived(url.replace(/^https?:\/\//, ''));

	let copied = $state(false);

	async function copy() {
		try {
			await navigator.clipboard.writeText(url);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			toast.error('Could not copy to clipboard');
		}
	}
</script>

<div class="flex min-w-0 items-center gap-1 rounded-lg border py-1 pr-1 pl-3">
	<code class="text-muted-foreground min-w-0 flex-1 truncate text-xs" title={url}>{display}</code>
	<Button variant="ghost" size="sm" aria-label="Copy link" title="Copy link" onclick={copy}>
		{#if copied}
			<Check class="size-3.5" />
		{:else}
			<Copy class="size-3.5" />
		{/if}
	</Button>
	{#if onRotate}
		<Button
			variant="ghost"
			size="sm"
			aria-label="Issue a new link"
			title="Issue a new link"
			onclick={onRotate}
		>
			<RefreshCw class="size-3.5" />
		</Button>
	{/if}
</div>
