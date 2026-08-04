<script lang="ts">
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';

	// A plain text box asking for "max total size" gives no clue what to type, so
	// this is a number plus an explicit unit. bytes is the bound value; 0 means no
	// limit, matching the convention used everywhere else.
	let {
		bytes = $bindable(0),
		id = 'size',
		label = 'Limit'
	}: { bytes?: number; id?: string; label?: string } = $props();

	const UNITS = [
		{ value: '1000000', label: 'MB' },
		{ value: '1000000000', label: 'GB' },
		{ value: '1000000000000', label: 'TB' }
	];

	function initialUnit(n: number) {
		if (n >= 1e12) return '1000000000000';
		if (n >= 1e9) return '1000000000';
		return '1000000';
	}

	let unit = $state(initialUnit(bytes));
	let amount = $state(bytes > 0 ? String(bytes / Number(initialUnit(bytes))) : '');

	const unitLabel = $derived(UNITS.find((u) => u.value === unit)?.label ?? 'MB');

	// One-way sync out: editing either control recomputes bytes. Deliberately not
	// syncing back in, which would fight the user mid-typing.
	$effect(() => {
		const n = Number(amount);
		bytes =
			amount.trim() === '' || !Number.isFinite(n) || n <= 0 ? 0 : Math.round(n * Number(unit));
	});
</script>

<div class="grid gap-1.5">
	<Label for={id}>{label}</Label>
	<div class="flex gap-2">
		<Input {id} bind:value={amount} inputmode="decimal" placeholder="No limit" class="flex-1" />
		<Select.Root type="single" bind:value={unit}>
			<Select.Trigger class="w-24">{unitLabel}</Select.Trigger>
			<Select.Content>
				{#each UNITS as u (u.value)}
					<Select.Item value={u.value} label={u.label}>{u.label}</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	</div>
</div>
