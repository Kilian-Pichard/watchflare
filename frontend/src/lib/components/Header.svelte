<script>
    import {
        mobileMenuOpen,
        sidebarCollapsed,
        toggleSidebarWithTransition,
        sidebarTransitioning,
    } from "$lib/stores/sidebar";
    import { uiStore } from "$lib/stores";

    const { title, showAlerts = false, alertCount = 0 } = $props();

    function toggleMenu() {
        mobileMenuOpen.update((val) => !val);
    }

    function toggleLeftSidebar() {
        toggleSidebarWithTransition();
    }

    function toggleAlerts() {
        uiStore.toggleRightSidebar();
    }
</script>

<header
    class="fixed left-0 right-0 top-0 z-30 h-fit py-4 px-2 md:px-4 bg-transparent {$sidebarCollapsed
        ? 'lg:left-20'
        : 'lg:left-64'} {$sidebarTransitioning
        ? 'transition-[left] duration-300 ease-in-out'
        : ''}"
>
    <div
        class="flex h-16 items-center justify-between px-4 bg-sidebar rounded-lg border"
    >
        <!-- Left: Mobile burger + Desktop left sidebar toggle -->
        <div class="flex items-center gap-2">
            <!-- Burger button (mobile only) -->
            <button
                onclick={toggleMenu}
                class="flex h-9 w-9 items-center justify-center rounded-lg text-foreground transition-colors hover:bg-muted lg:hidden"
                aria-label="Toggle menu"
            >
                {#if $mobileMenuOpen}
                    <svg
                        class="h-5 w-5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M6 18L18 6M6 6l12 12"
                        />
                    </svg>
                {:else}
                    <svg
                        class="h-5 w-5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M4 6h16M4 12h16M4 18h16"
                        />
                    </svg>
                {/if}
            </button>

            <!-- Left sidebar toggle (desktop only) -->
            <button
                onclick={toggleLeftSidebar}
                class="hidden lg:flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label={$sidebarCollapsed
                    ? "Expand sidebar"
                    : "Collapse sidebar"}
            >
                <svg
                    class="h-5 w-5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    stroke-width="2"
                >
                    <rect x="3" y="3" width="18" height="18" rx="2" />
                    <path d="M9 3v18" />
                </svg>
            </button>
        </div>

        <!-- Center: Title / Logo -->
        {#if title}
            <h1 class="text-base font-semibold text-foreground">{title}</h1>
        {:else}
            <span class="text-lg font-bold text-primary">Watchflare</span>
        {/if}

        <!-- Right: Alerts button or spacer -->
        <div class="flex items-center gap-2">
            {#if showAlerts}
                <button
                    onclick={toggleAlerts}
                    class="relative flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                    aria-label="Toggle alerts"
                >
                    <!-- Bell icon -->
                    <svg
                        class="h-5 w-5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        stroke-width="2"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
                        />
                    </svg>
                    {#if alertCount > 0}
                        <span
                            class="absolute top-1 right-1 h-2.5 w-2.5 rounded-full bg-destructive"
                        ></span>
                    {/if}
                </button>
            {:else}
                <div class="w-9"></div>
            {/if}
        </div>
    </div>
</header>
