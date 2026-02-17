<script>
	import { onMount, onDestroy } from 'svelte';
	import * as api from '$lib/api.js';
	import { logger } from '$lib/utils';

	export let server;
	export let token;
	export let agentKey;
	export let backendHost;

	let selectedOS = 'linux';
	let copied = false;
	let copyTimeout;
	let serverStatus = server.status;
	let pollInterval;

	// Instructions for each OS
	const quickInstallCmd = `curl -sSL https://get.watchflare.io/ | sudo bash -s -- \\
  --token ${token} \\
  --host ${backendHost} \\
  --port 50051`;

	const linuxCmd = `curl -sSL https://get.watchflare.io/linux | sudo bash -s -- \\
  --token ${token} \\
  --host ${backendHost} \\
  --port 50051`;

	const macosCmd = `curl -sSL https://get.watchflare.io/macos | sudo bash -s -- \\
  --token ${token} \\
  --host ${backendHost} \\
  --port 50051`;

	function copyToClipboard(text) {
		navigator.clipboard.writeText(text);
		copied = true;

		if (copyTimeout) clearTimeout(copyTimeout);
		copyTimeout = setTimeout(() => {
			copied = false;
		}, 2000);
	}

	// Poll server status every 5 seconds
	async function pollServerStatus() {
		try {
			const response = await api.getServer(server.id);
			serverStatus = response.status;

			// If server is online, stop polling and show celebration
			if (serverStatus === 'online') {
				clearInterval(pollInterval);
			}
		} catch (err) {
			logger.error('Failed to poll server status:', err);
		}
	}

	onMount(() => {
		// Start polling if server is not yet online
		if (serverStatus !== 'online') {
			pollInterval = setInterval(pollServerStatus, 5000);
		}
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
		if (copyTimeout) clearTimeout(copyTimeout);
	});
</script>

{#if serverStatus === 'online'}
	<!-- Success State -->
	<div class="success-banner">
		<div class="success-content">
			<span class="success-icon">🎉</span>
			<div>
				<h3>Agent Connected Successfully!</h3>
				<p>Your server is now online and sending heartbeats</p>
			</div>
		</div>
	</div>
{:else}
	<!-- Installation Instructions -->
	<div class="install-section">
		<h3>📦 Installation Instructions</h3>

		<!-- Quick Install (Recommended) -->
		<div class="install-option">
			<div class="option-header">
				<span class="option-badge recommended">Recommended</span>
				<h4>Quick Install (Auto-detect OS)</h4>
			</div>

			<p class="option-description">
				Run this command on your server. It will automatically detect your operating system and
				install the Watchflare agent:
			</p>

			<div class="code-block-wrapper">
				<pre class="code-block">{quickInstallCmd}</pre>
				<button class="copy-btn-code" on:click={() => copyToClipboard(quickInstallCmd)}>
					{copied ? '✓ Copied!' : 'Copy'}
				</button>
			</div>

			<div class="features-list">
				<span class="feature">✓ Automatically detects your OS</span>
				<span class="feature">✓ Downloads and installs agent</span>
				<span class="feature">✓ Registers automatically</span>
				<span class="feature">✓ Starts monitoring service</span>
			</div>
		</div>

		<!-- Platform-Specific (Advanced) -->
		<details class="advanced-section">
			<summary>Platform-Specific Installation (Advanced)</summary>

			<div class="tabs">
				<button
					class="tab"
					class:active={selectedOS === 'linux'}
					on:click={() => (selectedOS = 'linux')}
				>
					🐧 Linux
				</button>
				<button
					class="tab"
					class:active={selectedOS === 'macos'}
					on:click={() => (selectedOS = 'macos')}
				>
					🍎 macOS
				</button>
				<button class="tab" class:active={selectedOS === 'windows'} disabled>
					🪟 Windows <span class="badge-soon">Soon</span>
				</button>
				<button class="tab" class:active={selectedOS === 'docker'} disabled>
					🐳 Docker <span class="badge-soon">Soon</span>
				</button>
			</div>

			<div class="tab-content">
				{#if selectedOS === 'linux'}
					<div class="os-instructions">
						<h5>🐧 Linux Installation</h5>

						<div class="code-block-wrapper">
							<pre class="code-block">{linuxCmd}</pre>
							<button class="copy-btn-code" on:click={() => copyToClipboard(linuxCmd)}>
								{copied ? '✓ Copied!' : 'Copy'}
							</button>
						</div>

						<div class="supported-list">
							<p class="supported-title">Supported distributions:</p>
							<div class="supported-items">
								<span>✓ Ubuntu 18.04+</span>
								<span>✓ Debian 10+</span>
								<span>✓ CentOS 7+</span>
								<span>✓ RHEL 7+</span>
								<span>✓ Fedora 30+</span>
								<span>✓ Amazon Linux 2</span>
							</div>
						</div>
					</div>
				{:else if selectedOS === 'macos'}
					<div class="os-instructions">
						<h5>🍎 macOS Installation</h5>

						<div class="code-block-wrapper">
							<pre class="code-block">{macosCmd}</pre>
							<button class="copy-btn-code" on:click={() => copyToClipboard(macosCmd)}>
								{copied ? '✓ Copied!' : 'Copy'}
							</button>
						</div>

						<div class="supported-list">
							<p class="supported-title">Supported versions:</p>
							<div class="supported-items">
								<span>✓ macOS 11 (Big Sur) and later</span>
								<span>✓ Intel and Apple Silicon (M1/M2/M3)</span>
							</div>
						</div>
					</div>
				{/if}
			</div>
		</details>

		<!-- What Happens Next -->
		<div class="info-box">
			<h4>📖 What happens next?</h4>
			<ol>
				<li>The agent will register with this server</li>
				<li>
					Status will change from <span class="status-pending">"pending"</span> to
					<span class="status-online">"online"</span>
				</li>
				<li>You'll start receiving metrics and heartbeats</li>
				<li>This page will update automatically when connected</li>
			</ol>

			{#if serverStatus === 'pending'}
				<div class="waiting-indicator">
					<div class="spinner"></div>
					<span>Waiting for agent to connect...</span>
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.success-banner {
		background: linear-gradient(135deg, #48bb78 0%, #38a169 100%);
		color: white;
		padding: 2rem;
		border-radius: 12px;
		margin-bottom: 2rem;
		box-shadow: 0 4px 6px rgba(72, 187, 120, 0.3);
	}

	.success-content {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.success-icon {
		font-size: 3rem;
	}

	.success-banner h3 {
		margin: 0 0 0.25rem 0;
		font-size: 1.5rem;
		font-weight: 600;
	}

	.success-banner p {
		margin: 0;
		opacity: 0.9;
	}

	.install-section {
		margin-top: 2rem;
	}

	.install-section h3 {
		margin: 0 0 1.5rem 0;
		color: #1a202c;
		font-size: 1.25rem;
	}

	.install-option {
		background: white;
		border: 2px solid #e2e8f0;
		border-radius: 8px;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
	}

	.option-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 0.75rem;
	}

	.option-badge {
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.option-badge.recommended {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
	}

	.option-header h4 {
		margin: 0;
		color: #1a202c;
		font-size: 1rem;
	}

	.option-description {
		margin: 0 0 1rem 0;
		color: #4a5568;
		font-size: 0.875rem;
		line-height: 1.5;
	}

	.code-block-wrapper {
		position: relative;
		margin-bottom: 1rem;
	}

	.code-block {
		background: #1a202c;
		color: #e2e8f0;
		padding: 1rem;
		border-radius: 6px;
		overflow-x: auto;
		font-family: 'Monaco', 'Courier New', monospace;
		font-size: 0.875rem;
		margin: 0;
		line-height: 1.6;
	}

	.copy-btn-code {
		position: absolute;
		top: 0.5rem;
		right: 0.5rem;
		padding: 0.5rem 0.75rem;
		background: #2d3748;
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.copy-btn-code:hover {
		background: #4a5568;
	}

	.features-list {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 0.5rem;
	}

	.feature {
		color: #48bb78;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.advanced-section {
		background: #f7fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
	}

	.advanced-section summary {
		cursor: pointer;
		font-weight: 600;
		color: #4a5568;
		list-style: none;
		user-select: none;
	}

	.advanced-section summary::-webkit-details-marker {
		display: none;
	}

	.advanced-section summary::before {
		content: '▶';
		display: inline-block;
		margin-right: 0.5rem;
		transition: transform 0.2s;
	}

	.advanced-section[open] summary::before {
		transform: rotate(90deg);
	}

	.tabs {
		display: flex;
		gap: 0.5rem;
		margin-top: 1rem;
		margin-bottom: 1rem;
		border-bottom: 2px solid #e2e8f0;
	}

	.tab {
		padding: 0.75rem 1rem;
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		margin-bottom: -2px;
		cursor: pointer;
		font-weight: 500;
		color: #718096;
		transition: all 0.2s;
	}

	.tab:not(:disabled):hover {
		color: #667eea;
	}

	.tab.active {
		color: #667eea;
		border-bottom-color: #667eea;
	}

	.tab:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.badge-soon {
		font-size: 0.7rem;
		background: #fed7d7;
		color: #c53030;
		padding: 0.125rem 0.375rem;
		border-radius: 4px;
		margin-left: 0.25rem;
	}

	.tab-content {
		margin-top: 1rem;
	}

	.os-instructions h5 {
		margin: 0 0 1rem 0;
		color: #1a202c;
		font-size: 1rem;
	}

	.supported-list {
		margin-top: 1rem;
	}

	.supported-title {
		margin: 0 0 0.5rem 0;
		color: #4a5568;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.supported-items {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
		gap: 0.5rem;
	}

	.supported-items span {
		color: #48bb78;
		font-size: 0.875rem;
	}

	.info-box {
		background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
		border-left: 4px solid #667eea;
		padding: 1.5rem;
		border-radius: 6px;
	}

	.info-box h4 {
		margin: 0 0 1rem 0;
		color: #1a202c;
		font-size: 1rem;
	}

	.info-box ol {
		margin: 0 0 1rem 0;
		padding-left: 1.5rem;
		color: #4a5568;
	}

	.info-box li {
		margin-bottom: 0.5rem;
		font-size: 0.875rem;
		line-height: 1.5;
	}

	.status-pending {
		color: #ed8936;
		font-weight: 600;
	}

	.status-online {
		color: #48bb78;
		font-weight: 600;
	}

	.waiting-indicator {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 1rem;
		background: white;
		border-radius: 6px;
		margin-top: 1rem;
	}

	.spinner {
		width: 20px;
		height: 20px;
		border: 2px solid #e2e8f0;
		border-top-color: #667eea;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.waiting-indicator span {
		color: #4a5568;
		font-size: 0.875rem;
		font-weight: 500;
	}
</style>
