<script>
	import { Button } from '$lib/components/ui/button';
	import { TIME_RANGES } from '$lib/utils';
	import { updatePreferences } from '$lib/api';

	let { value = $bindable('24h'), onValueChange } = $props();

	async function handleSelect(timeRange) {
		value = timeRange;

		// Save to user preferences
		try {
			await updatePreferences(timeRange, undefined);
		} catch (err) {
			console.error('Failed to save time range preference:', err);
		}

		// Trigger callback if provided
		if (onValueChange) {
			onValueChange(timeRange);
		}
	}
</script>

<div class="flex gap-2 flex-wrap">
	{#each TIME_RANGES as range}
		<Button
			variant={value === range.value ? 'default' : 'outline'}
			size="sm"
			onclick={() => handleSelect(range.value)}
		>
			{range.label}
		</Button>
	{/each}
</div>
