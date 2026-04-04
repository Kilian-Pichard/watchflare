<script lang="ts">
    import { onMount } from "svelte";
    import {
        uiStore,
        sidebarCollapsed,
        sidebarTransitioning,
        userStore,
    } from "$lib/stores";
    import DesktopSidebar from "$lib/components/DesktopSidebar.svelte";
    import MobileSidebar from "$lib/components/MobileSidebar.svelte";
    import Header from "$lib/components/Header.svelte";
    import RightSidebar from "$lib/components/RightSidebar.svelte";

    const { children } = $props();

    let rightSidebarOpen = $derived($uiStore.rightSidebarOpen);
    let userReady = $state(false);

    onMount(async () => {
        try {
            await userStore.load();
            userReady = true;
        } catch {
            window.location.href = "/login";
        }
    });
</script>

{#if userReady}
    <div class="min-h-dvh bg-background">
        <Header />

        <DesktopSidebar />
        <MobileSidebar />

        <RightSidebar
            isOpen={rightSidebarOpen}
            onClose={() => uiStore.setRightSidebar(false)}
        />

        <main
            class="min-h-svh pt-26 p-4 sm:p-8 sm:pt-28 {$sidebarCollapsed
                ? 'lg:ml-20'
                : 'lg:ml-64'} {$sidebarTransitioning
                ? 'transition-[margin] duration-300 ease-in-out'
                : ''}"
        >
            {@render children()}
        </main>
    </div>
{/if}
