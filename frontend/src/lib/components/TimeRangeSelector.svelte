<script>
	import { TIME_RANGES, logger } from '$lib/utils';
	import { updatePreferences } from '$lib/api';

	let { value = $bindable('24h'), onValueChange } = $props();

	async function handleChange(e) {
		const timeRange = e.target.value;
		value = timeRange;

		// Save to user preferences
		try {
			await updatePreferences(timeRange, undefined);
		} catch (err) {
			logger.error('Failed to save time range preference:', err);
		}

		// Trigger callback if provided
		if (onValueChange) {
			onValueChange(timeRange);
		}
	}
</script>

<select
	{value}
	onchange={handleChange}
	class="rounded-lg border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
>
	{#each TIME_RANGES as range}
		<option value={range.value}>{range.label}</option>
	{/each}
</select>
