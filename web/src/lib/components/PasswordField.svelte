<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Switch from '$lib/components/ui/switch';
	import Eye from '@lucide/svelte/icons/eye';
	import EyeOff from '@lucide/svelte/icons/eye-off';

	let {
		enabled = $bindable(false),
		value = $bindable(''),
		id = 'password',
		label = 'Require a password'
	}: {
		enabled?: boolean;
		value?: string;
		id?: string;
		label?: string;
	} = $props();

	let revealed = $state(false);
</script>

<div class="grid gap-3">
	<div class="flex items-start gap-3">
		<Switch.Root {id} bind:checked={enabled} />
		<Label for={id}>{label}</Label>
	</div>

	{#if enabled}
		<div class="grid gap-1.5 pl-11">
			<div class="flex gap-2">
				<Input
					id="{id}-value"
					type={revealed ? 'text' : 'password'}
					bind:value
					autocomplete="new-password"
					placeholder="Choose a password"
				/>
				<Button
					type="button"
					variant="outline"
					size="icon"
					aria-label={revealed ? 'Hide password' : 'Show password'}
					onclick={() => (revealed = !revealed)}
				>
					{#if revealed}<EyeOff class="size-4" />{:else}<Eye class="size-4" />{/if}
				</Button>
			</div>
		</div>
	{/if}
</div>
