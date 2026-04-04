<script lang="ts">
    import { onMount } from "svelte";
    import { fade } from "svelte/transition";
    import { page } from "$app/stores";
    import { mobileMenuOpen } from "$lib/stores/sidebar";
    import { get } from "svelte/store";
    import { Settings, ChevronDown } from "lucide-svelte";
    import { navItems, settingsItems } from "$lib/navigation";
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

    let settingsOpen = $state($page.url.pathname.startsWith("/settings"));

    $effect(() => {
        if ($page.url.pathname.startsWith("/settings")) {
            settingsOpen = true;
        }
    });

    function isActive(href: string): boolean {
        if (href === "/") {
            return $page.url.pathname === "/";
        }
        return $page.url.pathname.startsWith(href);
    }

    function isSubActive(href: string): boolean {
        return $page.url.pathname === href;
    }

    function closeMobileMenu() {
        mobileMenuOpen.set(false);
    }
</script>

<!-- Mobile backdrop -->
{#if $mobileMenuOpen}
    <div
        transition:fade={{ duration: 200 }}
        class="fixed inset-0 z-30 bg-black/50 lg:hidden"
        role="presentation"
        onclick={closeMobileMenu}
    ></div>
{/if}

<div
    role="dialog"
    aria-label="Navigation menu"
    aria-modal="true"
    aria-hidden={!$mobileMenuOpen}
    inert={!$mobileMenuOpen}
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
                    onclick={closeMobileMenu}
                    aria-current={isActive(item.href) ? "page" : undefined}
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

            <!-- Settings group -->
            <div>
                <button
                    type="button"
                    aria-expanded={settingsOpen}
                    onclick={() => { settingsOpen = !settingsOpen; }}
                    class="w-full flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors text-surface-foreground hover:bg-surface-accent"
                >
                    <Settings class="h-5 w-5 shrink-0" />
                    <span class="flex-1 text-left">Settings</span>
                    <ChevronDown class="h-4 w-4 transition-transform duration-200 {settingsOpen ? 'rotate-180' : ''}" />
                </button>
                {#if settingsOpen}
                    <div class="ml-6 mt-1 mb-1 flex flex-col gap-0.5 border-l border-border pl-2">
                        {#each settingsItems as sub}
                            <a
                                href={sub.href}
                                onclick={closeMobileMenu}
                                aria-current={isSubActive(sub.href) ? "page" : undefined}
                                class="rounded-lg py-2.5 px-3 text-sm font-medium transition-colors {isSubActive(sub.href)
                                    ? 'bg-primary text-primary-foreground'
                                    : 'text-surface-foreground hover:bg-surface-accent'}"
                            >
                                {sub.label}
                            </a>
                        {/each}
                    </div>
                {/if}
            </div>
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
</div>
