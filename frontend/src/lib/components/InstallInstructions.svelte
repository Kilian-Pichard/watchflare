<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as api from '$lib/api.js';
	import { logger } from '$lib/utils';
	import { AGENT_STATUS_POLL_INTERVAL } from '$lib/constants';
	import type { Server } from '$lib/types';

	const { server, token, agentKey = '', backendHost }: {
		server: Server;
		token: string;
		agentKey?: string;
		backendHost: string;
	} = $props();

	let selectedOS = $state('linux');
	let copied = $state(false);
	let copyTimeout: ReturnType<typeof setTimeout> | null = $state(null);
	let polledStatus: string | null = $state(null);
	let serverStatus = $derived(polledStatus ?? server.status);
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	// Instructions for each OS
	let linuxCmd = $derived(`curl -sSL https://get.watchflare.io | sudo bash -s -- \\
  --token ${token} \\
  --host ${backendHost} \\
  --port 50051`);

	let macosCmd = $derived(`curl -sSL https://get.watchflare.io/brew | bash -s -- \\
  --token ${token} \\
  --host ${backendHost} \\
  --port 50051`);

	function handleCopy(text: string) {
		navigator.clipboard.writeText(text);
		copied = true;

		if (copyTimeout) clearTimeout(copyTimeout);
		copyTimeout = setTimeout(() => {
			copied = false;
		}, 2000);
	}

	async function pollServerStatus() {
		try {
			const response = await api.getServer(server.id);
			polledStatus = response.status;

			if (serverStatus === 'online') {
				if (pollInterval) clearInterval(pollInterval);
			}
		} catch (err) {
			logger.error('Failed to poll server status:', err);
		}
	}

	onMount(() => {
		if (serverStatus !== 'online') {
			pollInterval = setInterval(pollServerStatus, AGENT_STATUS_POLL_INTERVAL);
		}
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
		if (copyTimeout) clearTimeout(copyTimeout);
	});
</script>

{#if serverStatus === 'online'}
	<!-- Success State -->
	<div class="flex items-center gap-4 bg-success text-white p-8 rounded-xl mb-8 shadow-md">
		<span class="text-5xl">🎉</span>
		<div>
			<h3 class="text-2xl font-semibold mb-1">Agent Connected Successfully!</h3>
			<p class="opacity-90">Your server is now online and sending heartbeats</p>
		</div>
	</div>
{:else}
	<!-- Installation Instructions -->
	<div class="mt-8">
		<h3 class="text-xl font-semibold text-foreground mb-6">📦 Installation Instructions</h3>

		<!-- OS Tabs -->
		<div class="bg-card border-2 rounded-lg p-6 mb-6">
			<div class="flex gap-2 mb-6 border-b-2 border-border">
				<button
					class="px-4 py-3 bg-transparent border-b-2 -mb-0.5 font-medium transition-colors {selectedOS === 'linux'
						? 'text-primary border-primary'
						: 'text-muted-foreground border-transparent hover:text-primary'}"
					onclick={() => (selectedOS = 'linux')}
				>
					🐧 Linux
				</button>
				<button
					class="px-4 py-3 bg-transparent border-b-2 -mb-0.5 font-medium transition-colors {selectedOS === 'macos'
						? 'text-primary border-primary'
						: 'text-muted-foreground border-transparent hover:text-primary'}"
					onclick={() => (selectedOS = 'macos')}
				>
					🍎 macOS
				</button>
				<button
					class="px-4 py-3 bg-transparent border-b-2 -mb-0.5 font-medium opacity-50 cursor-not-allowed text-muted-foreground border-transparent"
					disabled
				>
					🪟 Windows <span class="text-[0.7rem] bg-destructive/10 text-destructive px-1.5 py-0.5 rounded ml-1">Soon</span>
				</button>
				<button
					class="px-4 py-3 bg-transparent border-b-2 -mb-0.5 font-medium opacity-50 cursor-not-allowed text-muted-foreground border-transparent"
					disabled
				>
					🐳 Docker <span class="text-[0.7rem] bg-destructive/10 text-destructive px-1.5 py-0.5 rounded ml-1">Soon</span>
				</button>
			</div>

			<div class="mt-4">
				{#if selectedOS === 'linux'}
					<div>
						<h5 class="text-base font-semibold text-foreground mb-4">🐧 Linux Installation</h5>

						<div class="relative mb-4">
							<pre class="bg-foreground text-background p-4 rounded-md font-mono text-sm leading-relaxed overflow-x-auto">{linuxCmd}</pre>
							<button
								class="absolute top-2 right-2 px-3 py-2 bg-muted-foreground/30 text-white rounded text-xs font-medium cursor-pointer transition-colors hover:bg-muted-foreground/50"
								onclick={() => handleCopy(linuxCmd)}
							>
								{copied ? '✓ Copied!' : 'Copy'}
							</button>
						</div>

						<div class="mt-4">
							<p class="text-sm font-semibold text-muted-foreground mb-2">Supported distributions:</p>
							<div class="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-2">
								<span class="text-success text-sm">✓ Ubuntu 18.04+</span>
								<span class="text-success text-sm">✓ Debian 10+</span>
								<span class="text-success text-sm">✓ CentOS 7+</span>
								<span class="text-success text-sm">✓ RHEL 7+</span>
								<span class="text-success text-sm">✓ Fedora 30+</span>
								<span class="text-success text-sm">✓ Amazon Linux 2</span>
							</div>
						</div>
					</div>
				{:else if selectedOS === 'macos'}
					<div>
						<h5 class="text-base font-semibold text-foreground mb-4">🍎 macOS Installation</h5>

						<div class="relative mb-4">
							<pre class="bg-foreground text-background p-4 rounded-md font-mono text-sm leading-relaxed overflow-x-auto">{macosCmd}</pre>
							<button
								class="absolute top-2 right-2 px-3 py-2 bg-muted-foreground/30 text-white rounded text-xs font-medium cursor-pointer transition-colors hover:bg-muted-foreground/50"
								onclick={() => handleCopy(macosCmd)}
							>
								{copied ? '✓ Copied!' : 'Copy'}
							</button>
						</div>

						<div class="mt-4">
							<p class="text-sm font-semibold text-muted-foreground mb-2">Supported versions:</p>
							<div class="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-2">
								<span class="text-success text-sm">✓ macOS 11 (Big Sur) and later</span>
								<span class="text-success text-sm">✓ Intel and Apple Silicon (M1/M2/M3)</span>
							</div>
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- What Happens Next -->
		<div class="bg-primary/5 border-l-4 border-primary p-6 rounded-md">
			<h4 class="text-base font-semibold text-foreground mb-4">📖 What happens next?</h4>
			<ol class="list-decimal pl-6 text-muted-foreground mb-4">
				<li class="mb-2 text-sm leading-relaxed">The agent will register with this server</li>
				<li class="mb-2 text-sm leading-relaxed">
					Status will change from <span class="text-warning font-semibold">"pending"</span> to
					<span class="text-success font-semibold">"online"</span>
				</li>
				<li class="mb-2 text-sm leading-relaxed">You'll start receiving metrics and heartbeats</li>
				<li class="mb-2 text-sm leading-relaxed">This page will update automatically when connected</li>
			</ol>

			{#if serverStatus === 'pending'}
				<div class="flex items-center gap-3 p-4 bg-card rounded-md mt-4">
					<div class="h-5 w-5 border-2 border-border border-t-primary rounded-full animate-spin"></div>
					<span class="text-sm font-medium text-muted-foreground">Waiting for agent to connect...</span>
				</div>
			{/if}
		</div>
	</div>
{/if}
