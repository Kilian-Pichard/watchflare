<script lang="ts">
    import { page } from '$app/stores';
    import ServerIncidentsSection from '$lib/components/server/ServerIncidentsSection.svelte';

    const serverId = $derived($page.params.id);
</script>

<ServerIncidentsSection {serverId} />
