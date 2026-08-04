<script lang="ts">
	import PasswordField from '$lib/components/PasswordField.svelte';
	import SizeField from '$lib/components/SizeField.svelte';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';
	import type { CreateShareDto } from '$lib/sdk';

	let {
		open = $bindable(false),
		onCreate
	}: { open?: boolean; onCreate: (payload: CreateShareDto) => Promise<void> } = $props();

	let title = $state('');
	let maxFiles = $state('');
	let maxBytes = $state(0);
	let passwordOn = $state(false);
	let password = $state('');
	let expiry = $state('0');
	let notifyUrl = $state('');
	let saving = $state(false);

	const EXPIRY_CHOICES = [
		{ value: '0', label: 'Never expires' },
		{ value: '7', label: 'In 7 days' },
		{ value: '30', label: 'In 30 days' },
		{ value: '90', label: 'In 90 days' }
	];
	const expiryLabel = $derived(
		EXPIRY_CHOICES.find((c) => c.value === expiry)?.label ?? 'Never expires'
	);

	async function submit() {
		const payload: CreateShareDto = { title: title.trim() || 'Upload Box', mode: 'upload_box' };

		if (maxFiles.trim()) payload.maxFiles = Number(maxFiles);
		if (maxBytes > 0) payload.maxBytes = maxBytes;
		if (expiry !== '0') payload.expiresInDays = Number(expiry);
		if (passwordOn && password) payload.password = password;
		if (notifyUrl.trim()) payload.notifyUrl = notifyUrl.trim();

		saving = true;
		try {
			await onCreate(payload);
			title = '';
			maxFiles = '';
			maxBytes = 0;
			password = '';
			passwordOn = false;
			notifyUrl = '';
			expiry = '0';
		} finally {
			saving = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[85vh] overflow-y-auto sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Collect files</Dialog.Title>
			<Dialog.Description>
				People can send you files without seeing what is inside. This cannot be changed to a send
				share later.
			</Dialog.Description>
		</Dialog.Header>

		<div class="grid gap-5">
			<div class="grid gap-1.5">
				<Label for="box-title">Name</Label>
				<Input id="box-title" bind:value={title} placeholder="Q3 Financials" />
			</div>

			<div class="grid gap-1.5">
				<Label for="box-files">Maximum number of files</Label>
				<Input id="box-files" bind:value={maxFiles} inputmode="numeric" placeholder="No limit" />
			</div>

			<SizeField bind:bytes={maxBytes} id="box-size" label="Maximum total size" />

			<div class="grid gap-1.5">
				<Label for="box-expiry">Expiry</Label>
				<Select.Root type="single" bind:value={expiry}>
					<Select.Trigger id="box-expiry">{expiryLabel}</Select.Trigger>
					<Select.Content>
						{#each EXPIRY_CHOICES as choice (choice.value)}
							<Select.Item value={choice.value} label={choice.label}>{choice.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>

			<PasswordField bind:enabled={passwordOn} bind:value={password} id="box-password" />

			<div class="grid gap-1.5">
				<Label for="box-notify">Notify me at</Label>
				<Input id="box-notify" bind:value={notifyUrl} placeholder="ntfy://ntfy.sh/my-topic" />
			</div>
		</div>

		<Dialog.Footer>
			<Dialog.Close type="button" class={buttonVariants({ variant: 'outline' })}
				>Cancel</Dialog.Close
			>
			<Button onclick={submit} disabled={saving}>Create Upload Box</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
