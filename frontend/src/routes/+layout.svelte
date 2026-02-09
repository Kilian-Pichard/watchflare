<script lang="ts">
	import '../app.css';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import Toast from '$lib/components/Toast.svelte';

	const { children } = $props();
</script>

<Tooltip.Provider>
	{@render children()}
</Tooltip.Provider>

<!-- Global toast notifications -->
<Toast />
