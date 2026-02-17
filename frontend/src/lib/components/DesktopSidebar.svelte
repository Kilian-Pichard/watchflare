<script>
    import { page } from "$app/stores";
    import {
        sidebarCollapsed,
        sidebarTransitioning,
    } from "$lib/stores/sidebar";
    import { Home, Server, Settings, LogOut } from "lucide-svelte";
    import SSEStatusBadge from "./SSEStatusBadge.svelte";

    const { onLogout } = $props();

    const transitioning = $derived($sidebarTransitioning);
    const collapsed = $derived($sidebarCollapsed);
    const transitionClass = $derived(transitioning ? 'transition-all duration-300 ease-in-out' : '');
    const textClass = $derived(collapsed
        ? `max-w-0 min-w-0 ml-0 opacity-0 ${transitionClass}`
        : `max-w-48 ml-3 opacity-100 ${transitionClass}`
    );

    const navItems = [
        { href: "/", label: "Dashboard", icon: Home },
        { href: "/servers", label: "Servers", icon: Server },
        { href: "/settings", label: "Settings", icon: Settings },
    ];

    function isActive(href) {
        if (href === "/") {
            return $page.url.pathname === "/";
        }
        return $page.url.pathname.startsWith(href);
    }
</script>

<aside
    class="fixed left-0 top-0 z-40 py-4 pl-4 hidden lg:block h-screen bg-transparent {collapsed
        ? 'w-20'
        : 'w-64'} {transitioning
        ? 'transition-[width] duration-300 ease-in-out'
        : ''}"
>
    <div
        class="flex h-full flex-col overflow-hidden bg-sidebar rounded-2xl border"
    >
        <!-- Logo -->
        <div class="flex h-16 items-center border-b px-[11px]">
            <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary text-lg font-bold text-primary-foreground">W</span>
            <span class="text-lg font-semibold text-foreground whitespace-nowrap overflow-hidden {textClass}">Watchflare</span>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 flex flex-col gap-1 p-2">
            {#each navItems as item}
                {@const Icon = item.icon}
                <a
                    href={item.href}
                    class="flex items-center rounded-lg py-[13px] px-[13px] text-sm font-medium transition-colors {isActive(item.href)
                        ? 'bg-primary text-primary-foreground'
                        : 'text-sidebar-foreground hover:bg-sidebar-accent'}"
                    title={item.label}
                >
                    <Icon class="h-5 w-5 flex-shrink-0" />
                    <span class="whitespace-nowrap overflow-hidden {textClass}">{item.label}</span>
                </a>
            {/each}
        </nav>

        <!-- SSE Connection Status + Logout -->
        <div class="border-t">
            <!-- SSE Status Badge -->
            <div class="px-2 pt-3 pb-1">
                <SSEStatusBadge {textClass} />
            </div>

            <!-- Logout Button -->
            <div class="px-2 pb-3">
                <button
                    onclick={onLogout}
                    class="flex w-full items-center rounded-lg py-[13px] px-[13px] text-sm font-medium text-destructive transition-colors hover:bg-destructive/10"
                    title="Logout"
                >
                    <LogOut class="h-5 w-5 flex-shrink-0" />
                    <span class="whitespace-nowrap overflow-hidden {textClass}">Logout</span>
                </button>
            </div>
        </div>
    </div>
</aside>
