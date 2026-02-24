<script lang="ts">
	import { goto } from '$app/navigation';
	import { currentUser, authActions } from '$lib/stores';
	import { userStore } from '$lib/stores/user';
	import type { Theme } from '$lib/types';
	import { User, Settings, LogOut, Sun, Moon, Monitor, Check } from 'lucide-svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';

	const {
		collapsed = false,
		textClass = '',
		onAction,
	}: {
		collapsed?: boolean;
		textClass?: string;
		onAction?: () => void;
	} = $props();

	const user = $derived($currentUser);
	const initials = $derived(
		user?.email
			? user.email.substring(0, 2).toUpperCase()
			: '??'
	);
	const currentTheme = $derived(user?.theme || 'system');

	const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
		{ value: 'light', label: 'Light', icon: Sun },
		{ value: 'dark', label: 'Dark', icon: Moon },
		{ value: 'system', label: 'System', icon: Monitor },
	];

	async function handleThemeChange(theme: Theme) {
		await userStore.updateTheme(theme);
	}

	function handleNavigate(path: string) {
		onAction?.();
		goto(path);
	}

	function handleLogout() {
		onAction?.();
		authActions.logout();
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				class="flex w-full items-center rounded-lg py-3.25 px-3.25 text-sm font-medium text-surface-foreground transition-colors hover:bg-surface-accent"
				title={user?.email || 'User menu'}
			>
				<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
					{initials}
				</span>
				<span class="whitespace-nowrap overflow-hidden text-left truncate {textClass}">
					{user?.email || 'User'}
				</span>
			</button>
		{/snippet}
	</DropdownMenu.Trigger>

	<DropdownMenu.Content side="top" align="start" class="mb-1">
		<a href="/user" onclick={() => onAction?.()} class="flex w-full cursor-pointer select-none items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground outline-none hover:bg-muted">
			<Settings class="h-4 w-4" />
			User Settings
		</a>

		<DropdownMenu.Separator />

		<div class="px-3 py-1.5 text-xs font-medium text-muted-foreground">Theme</div>
		{#each themeOptions as option}
			{@const Icon = option.icon}
			<DropdownMenu.Item onclick={() => handleThemeChange(option.value)}>
				<Icon class="h-4 w-4" />
				<span class="flex-1">{option.label}</span>
				{#if currentTheme === option.value}
					<Check class="h-4 w-4 text-primary" />
				{/if}
			</DropdownMenu.Item>
		{/each}

		<DropdownMenu.Separator />

		<DropdownMenu.Item onclick={handleLogout} class="text-destructive data-[highlighted]:text-destructive">
			<LogOut class="h-4 w-4" />
			Logout
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
