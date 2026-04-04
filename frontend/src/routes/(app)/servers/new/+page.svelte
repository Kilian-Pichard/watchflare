<script lang="ts">
	import { goto } from '$app/navigation';
	import * as api from '$lib/api.js';
	import { createServerSchema, validateForm } from '$lib/validation';
	import InstallInstructions from '$lib/components/InstallInstructions.svelte';

	let name = '';
	let configuredIP = '';
	let allowAnyIP = false;
	let error = '';
	let fieldErrors: Record<string, string> = {};
	let loading = false;
	let success = false;
	let createdServer = null;
	let token = '';
	let agentKey = '';
	let backendHost = '';

	async function handleSubmit(e) {
		e.preventDefault();
		error = '';
		fieldErrors = {};

		const result = validateForm(createServerSchema, { name, configuredIP, allowAnyIP });
		if (!result.success) {
			fieldErrors = result.errors;
			return;
		}

		loading = true;

		try {
			const response = await api.createServer(name, configuredIP, allowAnyIP);
			success = true;
			createdServer = response.server;
			token = response.token;
			agentKey = response.agent_key;
			backendHost = window.location.hostname;
		} catch (err) {
			error = err.message || 'Failed to create server';
		} finally {
			loading = false;
		}
	}

	function handleCopy(text) {
		navigator.clipboard.writeText(text);
	}
</script>

<svelte:head>
	<title>Add Server - Watchflare</title>
</svelte:head>

<!-- Back Link -->
<div class="mb-6">
	<a
		href="/servers"
		class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
	>
		<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
		</svg>
		Back to Servers
	</a>
</div>

{#if !success}
	<!-- Create Server Form -->
	<div class="max-w-2xl">
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-foreground">Add New Server</h1>
			<p class="text-sm text-muted-foreground mt-1">Configure a new server to monitor</p>
		</div>

		<div class="rounded-lg border bg-card p-4 sm:p-6">
			<form onsubmit={handleSubmit}>
				<!-- Server Name -->
				<div class="mb-4">
					<label for="name" class="block text-sm font-medium text-foreground mb-2">
						Server Name <span class="text-destructive">*</span>
					</label>
					<input
						id="name"
						type="text"
						bind:value={name}
						placeholder="e.g., web-server-01"
						aria-invalid={!!fieldErrors.name}
						aria-describedby={fieldErrors.name ? 'name-error' : undefined}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary {fieldErrors.name ? 'border-destructive' : ''}"
					/>
					{#if fieldErrors.name}<p id="name-error" class="mt-1 text-xs text-destructive">{fieldErrors.name}</p>{/if}
				</div>

				<!-- Configured IP -->
				<div class="mb-4">
					<label for="ip" class="block text-sm font-medium text-foreground mb-2">
						Configured IP Address <span class="text-destructive">*</span>
					</label>
					<input
						id="ip"
						type="text"
						bind:value={configuredIP}
						placeholder="e.g., 192.168.1.100"
						aria-invalid={!!fieldErrors.configuredIP}
						aria-describedby={fieldErrors.configuredIP ? 'configuredIP-error' : undefined}
						class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary {fieldErrors.configuredIP ? 'border-destructive' : ''}"
					/>
					{#if fieldErrors.configuredIP}<p id="configuredIP-error" class="mt-1 text-xs text-destructive">{fieldErrors.configuredIP}</p>{/if}
					<p class="mt-1 text-xs text-muted-foreground">
						The IP address you expect this server to connect from
					</p>
				</div>

				<!-- Allow Any IP -->
				<div class="mb-6">
					<label class="flex items-start gap-2 cursor-pointer">
						<input
							type="checkbox"
							bind:checked={allowAnyIP}
							class="mt-0.5 h-4 w-4 rounded border-gray-300"
						/>
						<div>
							<span class="text-sm font-medium text-foreground">Allow registration from any IP address</span>
							<p class="text-xs text-muted-foreground mt-0.5">
								If enabled, the server can register from any IP address
							</p>
						</div>
					</label>
				</div>

				<!-- Error Message -->
				{#if error}
					<div class="mb-4 rounded-lg border border-destructive bg-destructive/10 p-3">
						<p class="text-sm text-destructive">{error}</p>
					</div>
				{/if}

				<!-- Form Actions -->
				<div class="flex gap-3">
					<button
						type="submit"
						disabled={loading}
						class="flex-1 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{loading ? 'Creating...' : 'Create Server'}
					</button>
					<button
						type="button"
						onclick={() => goto('/servers')}
						class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
					>
						Cancel
					</button>
				</div>
			</form>
		</div>
	</div>
{:else}
	<!-- Success State -->
	<div class="max-w-3xl">
		<!-- Success Header -->
		<div class="mb-6 rounded-lg border border-success bg-success/10 p-4">
			<div class="flex items-start gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-full bg-success text-primary-foreground">
					<svg class="h-6 w-6" fill="currentColor" viewBox="0 0 20 20">
						<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
					</svg>
				</div>
				<div>
					<h2 class="text-lg font-semibold text-success">Server Created Successfully!</h2>
					<p class="text-sm text-muted-foreground mt-1">
						Server "{createdServer.name}" has been created with status: {createdServer.status}
					</p>
				</div>
			</div>
		</div>

		<!-- Tokens -->
		<div class="mb-6 rounded-lg border bg-card p-4 sm:p-6">
			<h3 class="text-base font-semibold text-foreground mb-4">Credentials</h3>

			<!-- Registration Token -->
			<div class="mb-4">
				<label class="block text-sm font-medium text-foreground mb-2">Registration Token</label>
				<div class="flex gap-2">
					<input
						type="text"
						readonly
						value={token}
						class="flex-1 rounded-lg border bg-muted px-3 py-2 font-mono text-xs text-foreground"
					/>
					<button
						onclick={() => handleCopy(token)}
						class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
					>
						Copy
					</button>
				</div>
				<p class="mt-2 text-xs font-medium text-warning">
					⚠️ Save this token securely. It won't be shown again!
				</p>
			</div>

			<!-- Agent Key -->
			<div>
				<label class="block text-sm font-medium text-foreground mb-2">Agent Key</label>
				<div class="flex gap-2">
					<input
						type="text"
						readonly
						value={agentKey}
						class="flex-1 rounded-lg border bg-muted px-3 py-2 font-mono text-xs text-foreground"
					/>
					<button
						onclick={() => handleCopy(agentKey)}
						class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
					>
						Copy
					</button>
				</div>
			</div>
		</div>

		<!-- Install Instructions -->
		<InstallInstructions server={createdServer} {token} {agentKey} {backendHost} />

		<!-- Actions -->
		<div class="mt-6 flex gap-3">
			<button
				onclick={() => goto(`/servers/${createdServer.id}`)}
				class="flex-1 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
			>
				View Server Details
			</button>
			<button
				onclick={() => goto('/servers')}
				class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
			>
				Back to List
			</button>
		</div>
	</div>
{/if}
