<script>
    import { onMount } from "svelte";
    import { fade } from "svelte/transition";
    import { page } from "$app/stores";
    import { mobileMenuOpen } from "$lib/stores/sidebar";
    import { get } from "svelte/store";
    import SSEStatusBadge from './SSEStatusBadge.svelte';

    const { onLogout } = $props();

    let wasOpenBeforeDesktop = false;

    onMount(() => {
        const mediaQuery = window.matchMedia("(min-width: 1024px)");

        const handleChange = (e) => {
            if (e.matches) {
                // Passage en desktop : sauvegarder l'état et fermer
                wasOpenBeforeDesktop = get(mobileMenuOpen);
                mobileMenuOpen.set(false);
            } else {
                // Retour en mobile : rouvrir si c'était ouvert
                if (wasOpenBeforeDesktop) {
                    // Petit délai pour que la transition se joue
                    setTimeout(() => {
                        mobileMenuOpen.set(true);
                    }, 50);
                }
            }
        };

        mediaQuery.addEventListener("change", handleChange);

        return () => {
            mediaQuery.removeEventListener("change", handleChange);
        };
    });

    const navItems = [
        {
            href: "/",
            label: "Dashboard",
            icon: "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6",
        },
        {
            href: "/servers",
            label: "Servers",
            icon: "M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01",
        },
        {
            href: "/settings",
            label: "Settings",
            icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z",
        },
    ];

    function isActive(href) {
        if (href === "/") {
            return $page.url.pathname === "/";
        }
        return $page.url.pathname.startsWith(href);
    }

    function closeMobileMenu() {
        mobileMenuOpen.set(false);
    }

    function handleNavClick() {
        closeMobileMenu();
    }
</script>

<!-- Mobile backdrop -->
{#if $mobileMenuOpen}
    <div
        transition:fade={{ duration: 200 }}
        class="fixed inset-0 z-30 bg-black/50 lg:hidden"
        onclick={closeMobileMenu}
    ></div>
{/if}

<aside
    class="fixed left-0 top-0 z-40 lg:hidden h-screen w-64 border-r bg-sidebar overflow-y-auto transition-transform duration-300 {$mobileMenuOpen
        ? 'translate-x-0'
        : '-translate-x-full'}"
>
    <div class="flex h-full flex-col">
        <!-- Logo -->
        <div class="flex h-16 items-center border-b justify-between px-6">
            <h1 class="text-xl font-semibold text-foreground">Watchflare</h1>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 space-y-1 p-4">
            {#each navItems as item}
                <a
                    href={item.href}
                    onclick={handleNavClick}
                    class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors {isActive(
                        item.href,
                    )
                        ? 'bg-primary text-primary-foreground'
                        : 'text-sidebar-foreground hover:bg-sidebar-accent'}"
                >
                    <svg
                        class="h-5 w-5 flex-shrink-0"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d={item.icon}
                        />
                    </svg>
                    <span>{item.label}</span>
                </a>
            {/each}
        </nav>

        <!-- SSE Connection Status + Logout -->
        <div class="border-t">
            <!-- SSE Status Badge -->
            <div class="px-4 pt-4 pb-2">
                <SSEStatusBadge />
            </div>

            <!-- Logout Button -->
            <div class="px-4 pb-4">
                <button
                    onclick={onLogout}
                    class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10"
                >
                    <svg
                        class="h-5 w-5 flex-shrink-0"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
                        />
                    </svg>
                    <span>Logout</span>
                </button>
            </div>
        </div>
    </div>
</aside>
