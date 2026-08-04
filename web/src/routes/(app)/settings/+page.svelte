<script lang="ts">
	import AccountSettings from '$lib/components/settings/AccountSettings.svelte';
	import UserSettings from '$lib/components/settings/UserSettings.svelte';
	import { Separator } from '$lib/components/ui/separator';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Settings — dropla</title></svelte:head>

<h1 class="text-2xl font-semibold tracking-tight">Settings</h1>
<p class="text-muted-foreground mt-1 mb-8 text-sm">
	Signed in as <span class="font-medium">{data.user.username}</span>{#if data.user.isAdmin}
		· administrator{/if}
</p>

<section class="mb-10">
	<h2 class="mb-4 text-lg font-semibold tracking-tight">Your account</h2>
	<AccountSettings />
</section>

{#if data.user.isAdmin}
	<Separator class="mb-10" />

	<section>
		<h2 class="mb-1 text-lg font-semibold tracking-tight">Users</h2>
		<UserSettings />
	</section>
{/if}
