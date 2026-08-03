<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import './app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>app</title>
</svelte:head>

<div class="mx-auto min-h-svh max-w-2xl px-5 py-12">
	{@render children()}
</div>
