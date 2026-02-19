<script lang="ts">
    import { goto } from "$app/navigation";
    import { logout } from "$lib/api";
    import { countAlerts, logger } from "$lib/utils";
    import {
        userStore,
        serversStore,
        servers,
        metricsStore,
        aggregatedStore,
        alertsStore,
        uiStore,
        sidebarCollapsed,
        sidebarTransitioning,
    } from "$lib/stores";
    import DesktopSidebar from "$lib/components/DesktopSidebar.svelte";
    import MobileSidebar from "$lib/components/MobileSidebar.svelte";
    import Header from "$lib/components/Header.svelte";
    import RightSidebar from "$lib/components/RightSidebar.svelte";

    const { children } = $props();

    let serversList = $derived($servers);
    let rightSidebarOpen = $derived($uiStore.rightSidebarOpen);
    let alertCount = $derived(countAlerts(serversList));

    async function handleLogout() {
        try {
            await logout();
            userStore.clear();
            serversStore.clear();
            metricsStore.clear();
            aggregatedStore.clear();
            alertsStore.clear();
            goto("/login");
        } catch (err) {
            logger.error("Logout failed:", err);
            goto("/login");
        }
    }
</script>

<div class="min-h-screen bg-background">
    <Header {alertCount} />

    <DesktopSidebar onLogout={handleLogout} />
    <MobileSidebar onLogout={handleLogout} />

    <RightSidebar
        servers={serversList}
        isOpen={rightSidebarOpen}
        onClose={() => uiStore.setRightSidebar(false)}
    />

    <main
        class="min-h-screen pt-26 p-4 sm:p-8 sm:pt-28 {$sidebarCollapsed
            ? 'lg:ml-20'
            : 'lg:ml-64'} {$sidebarTransitioning
            ? 'transition-[margin] duration-300 ease-in-out'
            : ''}"
    >
        {@render children()}
    </main>
</div>
