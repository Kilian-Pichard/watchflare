<script lang="ts">
	import * as Select from '$lib/components/ui/select';

	const {
		searchQuery,
		statusFilter,
		environmentFilter,
		onSearchInput,
		onStatusChange,
		onEnvironmentChange,
	}: {
		searchQuery: string;
		statusFilter: string;
		environmentFilter: string;
		onSearchInput: (e: Event) => void;
		onStatusChange: (value: string) => void;
		onEnvironmentChange: (value: string) => void;
	} = $props();

	const statusOptions = [
		{ value: '', label: 'All statuses' },
		{ value: 'online', label: 'Online' },
		{ value: 'offline', label: 'Offline' },
		{ value: 'pending', label: 'Pending' },
	];

	const environmentOptions = [
		{ value: '', label: 'All environments' },
		{ value: 'physical', label: 'Physical' },
		{ value: 'physical_with_containers', label: 'Physical + Containers' },
		{ value: 'vm', label: 'VM' },
		{ value: 'vm_with_containers', label: 'VM + Containers' },
		{ value: 'container', label: 'Container' },
	];

	const statusLabel = $derived(
		statusOptions.find((o) => o.value === statusFilter)?.label || 'All statuses'
	);
	const environmentLabel = $derived(
		environmentOptions.find((o) => o.value === environmentFilter)?.label || 'All environments'
	);
</script>

<div class="mb-4 flex flex-wrap items-center gap-3">
	<input
		type="text"
		placeholder="Search by name or hostname..."
		value={searchQuery}
		oninput={onSearchInput}
		class="rounded-lg border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 w-full sm:w-64"
	/>
	<Select.Root type="single" value={statusFilter} onValueChange={onStatusChange}>
		<Select.Trigger items={statusOptions.map((o) => o.label)}>
			<span>{statusLabel}</span>
		</Select.Trigger>
		<Select.Content>
			{#each statusOptions as option}
				<Select.Item value={option.value} label={option.label}>
					{option.label}
				</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
	<Select.Root type="single" value={environmentFilter} onValueChange={onEnvironmentChange}>
		<Select.Trigger items={environmentOptions.map((o) => o.label)}>
			<span>{environmentLabel}</span>
		</Select.Trigger>
		<Select.Content>
			{#each environmentOptions as option}
				<Select.Item value={option.value} label={option.label}>
					{option.label}
				</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
</div>
