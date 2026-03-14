<script lang="ts">
    import { onMount } from "svelte";
    import { fade } from "svelte/transition";
    import { page } from "$app/stores";
    import { mobileMenuOpen } from "$lib/stores/sidebar";
    import { get } from "svelte/store";
    import { Home, Server, Settings } from "lucide-svelte";
    import SSEStatusBadge from "./SSEStatusBadge.svelte";
    import UserMenuButton from "./UserMenuButton.svelte";

    let wasOpenBeforeDesktop = false;

    onMount(() => {
        const mediaQuery = window.matchMedia("(min-width: 1024px)");

        const handleChange = (e: MediaQueryListEvent) => {
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
        { href: "/", label: "Dashboard", icon: Home },
        { href: "/servers", label: "Servers", icon: Server },
        { href: "/settings", label: "Settings", icon: Settings },
    ];

    function isActive(href: string): boolean {
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

<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
<!-- Mobile backdrop -->
{#if $mobileMenuOpen}
    <div
        transition:fade={{ duration: 200 }}
        class="fixed inset-0 z-30 bg-black/50 lg:hidden"
        role="presentation"
        onclick={closeMobileMenu}
    ></div>
{/if}

<aside
    class="fixed left-0 top-0 z-40 py-4 pl-4 lg:hidden h-svh w-4/5 max-w-72 bg-transparent overflow-y-auto transition-transform duration-300 {$mobileMenuOpen
        ? 'translate-x-0'
        : '-translate-x-full'}"
>
    <div class="flex h-full flex-col bg-surface rounded-2xl border">
        <!-- Logo -->
        <div class="flex h-16 items-center border-b justify-between px-6">
            <h1 class="text-xl font-semibold text-foreground">Watchflare</h1>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 space-y-1 p-4">
            {#each navItems as item}
                {@const Icon = item.icon}
                <a
                    href={item.href}
                    onclick={handleNavClick}
                    class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors {isActive(
                        item.href,
                    )
                        ? 'bg-primary text-primary-foreground'
                        : 'text-surface-foreground hover:bg-surface-accent'}"
                >
                    <Icon class="h-5 w-5 shrink-0" />
                    <span>{item.label}</span>
                </a>
            {/each}
        </nav>

        <!-- SSE Connection Status + User Menu -->
        <div class="border-t">
            <!-- SSE Status Badge -->
            <div class="px-4 pt-4 pb-2">
                <SSEStatusBadge />
            </div>

            <!-- User Menu -->
            <div class="px-4 pb-4">
                <UserMenuButton collapsed={false} onAction={closeMobileMenu} />
            </div>
        </div>
    </div>
</aside>
