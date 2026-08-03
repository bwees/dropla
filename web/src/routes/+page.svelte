<script lang="ts">
	import { onMount } from 'svelte';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { getNotes, createNote, deleteNote, apiErrorMessage, type Note } from '$lib/sdk';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import * as Card from '$lib/components/ui/card';

	let notes = $state<Note[]>([]);
	let title = $state('');
	let body = $state('');
	let error = $state('');
	let loading = $state(true);
	let saving = $state(false);

	onMount(load);

	async function load() {
		try {
			notes = await getNotes();
		} catch (e) {
			error = apiErrorMessage(e);
		} finally {
			loading = false;
		}
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		if (!title.trim()) return;

		saving = true;
		error = '';
		try {
			const note = await createNote({ title: title.trim(), body: body.trim() });
			notes = [...notes, note];
			title = '';
			body = '';
		} catch (e) {
			error = apiErrorMessage(e);
		} finally {
			saving = false;
		}
	}

	async function remove(id: string) {
		error = '';
		try {
			await deleteNote(id);
			notes = notes.filter((note) => note.id !== id);
		} catch (e) {
			error = apiErrorMessage(e);
		}
	}
</script>

<header class="mb-8">
	<h1 class="text-2xl font-semibold tracking-tight">Notes</h1>
	<p class="text-muted-foreground text-sm">
		A Go + SvelteKit template, wired together with a generated OpenAPI client.
	</p>
</header>

<Card.Root class="mb-8">
	<Card.Header>
		<Card.Title>New note</Card.Title>
	</Card.Header>
	<Card.Content>
		<form class="grid gap-4" onsubmit={add}>
			<div class="grid gap-2">
				<Label for="title">Title</Label>
				<Input id="title" placeholder="Title" bind:value={title} />
			</div>
			<div class="grid gap-2">
				<Label for="body">Body</Label>
				<Textarea id="body" placeholder="Write something..." rows={3} bind:value={body} />
			</div>
			{#if error}
				<p class="text-destructive text-sm">{error}</p>
			{/if}
			<Button type="submit" disabled={saving || !title.trim()} class="justify-self-start">
				{saving ? 'Adding...' : 'Add note'}
			</Button>
		</form>
	</Card.Content>
</Card.Root>

{#if loading}
	<p class="text-muted-foreground text-center text-sm">Loading...</p>
{:else if notes.length === 0}
	<p class="text-muted-foreground text-center text-sm">No notes yet. Add one above.</p>
{:else}
	<ul class="grid gap-3">
		{#each notes as note (note.id)}
			<li>
				<Card.Root>
					<Card.Content class="flex items-start justify-between gap-4">
						<div class="min-w-0">
							<h3 class="font-medium">{note.title}</h3>
							{#if note.body}
								<p class="text-muted-foreground mt-1 text-sm whitespace-pre-wrap">{note.body}</p>
							{/if}
						</div>
						<Button
							variant="ghost"
							size="icon"
							aria-label="Delete note"
							onclick={() => remove(note.id)}
						>
							<Trash2 class="size-4" />
						</Button>
					</Card.Content>
				</Card.Root>
			</li>
		{/each}
	</ul>
{/if}
