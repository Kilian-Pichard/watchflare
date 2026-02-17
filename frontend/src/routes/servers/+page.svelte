<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { goto } from "$app/navigation";
    import { logout } from "$lib/api.js";
    import * as api from "$lib/api.js";
    import { toasts } from "$lib/stores/toasts";
    import { sidebarCollapsed, sidebarTransitioning } from "$lib/stores/sidebar";
    import { getStatusClass, formatRelativeTime, logger } from "$lib/utils";
    import type { Server, SSEEvent } from "$lib/types";
    import { sseStore } from "$lib/stores/sse";
    import DesktopSidebar from "$lib/components/DesktopSidebar.svelte";
    import MobileSidebar from "$lib/components/MobileSidebar.svelte";
    import Header from "$lib/components/Header.svelte";

    const PER_PAGE = 20;

    let servers: Server[] = [];
    let total = 0;
    let page = 1;
    let initialLoading = true;
    let loading = false;
    let error = "";
    let showDeleteConfirm = false;
    let serverToDelete: Server | null = null;
    let sseUnsubscribe: (() => void) | null = null;

    // Sort state
    let sortColumn = "created_at";
    let sortOrder: "asc" | "desc" = "desc";

    // Filter state
    let searchQuery = "";
    let statusFilter = "";
    let environmentFilter = "";
    let searchTimeout: ReturnType<typeof setTimeout> | null = null;

    $: totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

    async function loadPage(p: number) {
        loading = true;
        error = "";
        try {
            const response = await api.listServers({
                page: p,
                perPage: PER_PAGE,
                sort: sortColumn,
                order: sortOrder,
                status: statusFilter || undefined,
                search: searchQuery || undefined,
                environment: environmentFilter || undefined,
            });
            servers = response.servers || [];
            total = response.total || 0;
            page = p;
        } catch (err: unknown) {
            error = err instanceof Error ? err.message : "Failed to load servers";
        } finally {
            loading = false;
            initialLoading = false;
        }
    }

    function handleSort(column: string) {
        if (sortColumn === column) {
            sortOrder = sortOrder === "asc" ? "desc" : "asc";
        } else {
            sortColumn = column;
            sortOrder = "asc";
        }
        loadPage(page);
    }

    function handleSearchInput(e: Event) {
        const value = (e.target as HTMLInputElement).value;
        searchQuery = value;
        if (searchTimeout) clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            loadPage(1);
        }, 300);
    }

    function handleStatusChange(e: Event) {
        statusFilter = (e.target as HTMLSelectElement).value;
        loadPage(1);
    }

    function handleEnvironmentChange(e: Event) {
        environmentFilter = (e.target as HTMLSelectElement).value;
        loadPage(1);
    }

    async function dismissReactivation(serverId: string) {
        try {
            await api.dismissReactivation(serverId);
            await loadPage(page);
        } catch (err) {
            logger.error("Failed to dismiss reactivation:", err);
        }
    }

    async function handleLogout() {
        try {
            await logout();
            goto("/login");
        } catch (err) {
            logger.error("Logout failed:", err);
            goto("/login");
        }
    }

    function handleSSEMessage(event: SSEEvent) {
        if (event.type === "server_update") {
            const update = event.data;

            if (update.reactivated && update.hostname) {
                toasts.add(
                    `Agent "${update.hostname}" was reactivated (same physical server detected via UUID)`,
                    "info",
                    8000,
                );
            }

            const serverIndex = servers.findIndex((s) => s.id === update.id);
            if (serverIndex !== -1) {
                servers[serverIndex] = {
                    ...servers[serverIndex],
                    status: update.status,
                    ip_address_v4: update.ip_address_v4,
                    ip_address_v6: update.ip_address_v6,
                    configured_ip: update.configured_ip,
                    ignore_ip_mismatch: update.ignore_ip_mismatch,
                    last_seen: update.last_seen,
                };
                servers = [...servers];
            }
        }
    }

    onMount(async () => {
        await loadPage(1);
        sseUnsubscribe = sseStore.connect(handleSSEMessage);
    });

    onDestroy(() => {
        if (sseUnsubscribe) {
            sseUnsubscribe();
        }
        if (searchTimeout) clearTimeout(searchTimeout);
    });

    function hasIPMismatch(server: Server) {
        return (
            server.configured_ip &&
            server.ip_address_v4 &&
            server.configured_ip !== server.ip_address_v4 &&
            !server.ignore_ip_mismatch
        );
    }

    function openDeleteModal(server: Server, e: Event) {
        e.stopPropagation();
        serverToDelete = server;
        showDeleteConfirm = true;
    }

    function cancelDelete() {
        showDeleteConfirm = false;
        serverToDelete = null;
    }

    async function handleDelete() {
        if (!serverToDelete) return;

        try {
            await api.deleteServer(serverToDelete.id);
            await loadPage(page);
            showDeleteConfirm = false;
            serverToDelete = null;
        } catch (err: unknown) {
            error = err instanceof Error ? err.message : "Failed to delete server";
            showDeleteConfirm = false;
            serverToDelete = null;
        }
    }
</script>

<svelte:head>
    <title>Servers - Watchflare</title>
</svelte:head>

<svelte:window onkeydown={e => e.key === 'Escape' && showDeleteConfirm && cancelDelete()} />

<div class="min-h-screen bg-background">
    <!-- Header -->
    <Header title="Servers" />

    <!-- Desktop Sidebar -->
    <DesktopSidebar onLogout={handleLogout} />

    <!-- Mobile Sidebar -->
    <MobileSidebar onLogout={handleLogout} />

    <main
        class="min-h-screen pt-16 p-4 md:p-8 md:pt-20 {$sidebarCollapsed
            ? 'lg:ml-20'
            : 'lg:ml-64'} {$sidebarTransitioning ? 'transition-[margin] duration-300 ease-in-out' : ''}"
    >
        <!-- Header -->
        <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
                <h1 class="text-xl sm:text-2xl font-semibold text-foreground">Servers</h1>
                <p class="text-sm text-muted-foreground mt-1">
                    Manage your monitored servers
                </p>
            </div>
            <button
                onclick={() => goto("/servers/new")}
                class="self-start sm:self-auto rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            >
                Add Server
            </button>
        </div>

        <!-- Filters -->
        <div class="mb-4 flex flex-wrap items-center gap-3">
            <input
                type="text"
                placeholder="Search by name or hostname..."
                value={searchQuery}
                oninput={handleSearchInput}
                class="rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 w-full sm:w-64"
            />
            <select
                value={statusFilter}
                onchange={handleStatusChange}
                class="rounded-lg border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
            >
                <option value="">All statuses</option>
                <option value="online">Online</option>
                <option value="offline">Offline</option>
                <option value="pending">Pending</option>
            </select>
            <select
                value={environmentFilter}
                onchange={handleEnvironmentChange}
                class="rounded-lg border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
            >
                <option value="">All environments</option>
                <option value="physical">Physical</option>
                <option value="physical_with_containers">Physical + Containers</option>
                <option value="vm">VM</option>
                <option value="vm_with_containers">VM + Containers</option>
                <option value="container">Container</option>
            </select>
        </div>

        {#if initialLoading}
            <div class="flex items-center justify-center py-20">
                <p class="text-muted-foreground">Loading servers...</p>
            </div>
        {:else if error}
            <div
                class="rounded-lg border border-destructive bg-destructive/10 p-4"
            >
                <p class="text-sm text-destructive">{error}</p>
            </div>
        {:else if servers.length === 0 && page === 1 && !searchQuery && !statusFilter && !environmentFilter}
            <div
                class="flex flex-col items-center justify-center rounded-lg border bg-card py-20 text-center"
            >
                <svg
                    class="h-12 w-12 text-muted-foreground/50 mb-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="1.5"
                        d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                    />
                </svg>
                <h3 class="text-lg font-medium text-foreground mb-2">
                    No servers configured yet
                </h3>
                <p class="text-sm text-muted-foreground mb-6">
                    Add your first server to start monitoring
                </p>
                <button
                    onclick={() => goto("/servers/new")}
                    class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
                >
                    Add Your First Server
                </button>
            </div>
        {:else if servers.length === 0}
            <div
                class="flex flex-col items-center justify-center rounded-lg border bg-card py-12 text-center"
            >
                <p class="text-sm text-muted-foreground">No servers match your filters</p>
            </div>
        {:else}
            <div class="rounded-lg border bg-card overflow-x-auto">
                    <table class="w-full min-w-[800px]">
                        <thead>
                            <tr class="border-b bg-muted/30">
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
                                    onclick={() => handleSort("name")}
                                >
                                    <span class="inline-flex items-center gap-1">
                                        Name
                                        {#if sortColumn === "name"}
                                            <svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
                                                {#if sortOrder === "asc"}
                                                    <path d="M6 2l4 5H2z" />
                                                {:else}
                                                    <path d="M6 10l4-5H2z" />
                                                {/if}
                                            </svg>
                                        {/if}
                                    </span>
                                </th>
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
                                    onclick={() => handleSort("status")}
                                >
                                    <span class="inline-flex items-center gap-1">
                                        Status
                                        {#if sortColumn === "status"}
                                            <svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
                                                {#if sortOrder === "asc"}
                                                    <path d="M6 2l4 5H2z" />
                                                {:else}
                                                    <path d="M6 10l4-5H2z" />
                                                {/if}
                                            </svg>
                                        {/if}
                                    </span>
                                </th>
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
                                    onclick={() => handleSort("ip")}
                                >
                                    <span class="inline-flex items-center gap-1">
                                        IP Address
                                        {#if sortColumn === "ip"}
                                            <svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
                                                {#if sortOrder === "asc"}
                                                    <path d="M6 2l4 5H2z" />
                                                {:else}
                                                    <path d="M6 10l4-5H2z" />
                                                {/if}
                                            </svg>
                                        {/if}
                                    </span>
                                </th>
                                <th
                                    class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none hover:text-foreground transition-colors"
                                    onclick={() => handleSort("last_seen")}
                                >
                                    <span class="inline-flex items-center gap-1 justify-end w-full">
                                        Last Seen
                                        {#if sortColumn === "last_seen"}
                                            <svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
                                                {#if sortOrder === "asc"}
                                                    <path d="M6 2l4 5H2z" />
                                                {:else}
                                                    <path d="M6 10l4-5H2z" />
                                                {/if}
                                            </svg>
                                        {/if}
                                    </span>
                                </th>
                                <th
                                    class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider"
                                >
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-border">
                            {#each servers as server}
                                <tr
                                    onclick={() =>
                                        goto(`/servers/${server.id}`)}
                                    class="hover:bg-muted/20 transition-colors cursor-pointer"
                                >
                                    <td class="px-4 py-3.5">
                                        <div class="flex flex-col">
                                            <span
                                                class="font-medium text-foreground"
                                                >{server.name}</span
                                            >
                                            {#if server.hostname}
                                                <span
                                                    class="text-xs text-muted-foreground"
                                                    >{server.hostname}</span
                                                >
                                            {/if}
                                        </div>
                                    </td>
                                    <td class="px-4 py-3.5">
                                        <div class="flex items-center gap-2">
                                            <span
                                                class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(
                                                    server.status,
                                                )}"
                                            >
                                                <span
                                                    class="h-1.5 w-1.5 rounded-full {server.status ===
                                                    'online'
                                                        ? 'bg-success'
                                                        : 'bg-muted-foreground'}"
                                                ></span>
                                                {server.status}
                                            </span>
                                            {#if hasIPMismatch(server)}
                                                <span
                                                    class="inline-flex items-center text-warning"
                                                    title="IP mismatch detected"
                                                >
                                                    <svg
                                                        class="h-4 w-4"
                                                        fill="currentColor"
                                                        viewBox="0 0 20 20"
                                                    >
                                                        <path
                                                            fill-rule="evenodd"
                                                            d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                                                            clip-rule="evenodd"
                                                        />
                                                    </svg>
                                                </span>
                                            {/if}
                                            {#if server.reactivated_at}
                                                <span
                                                    class="inline-flex items-center gap-1 rounded-full border border-primary/20 bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
                                                    title="Agent was reactivated (same physical server via UUID)"
                                                >
                                                    <svg
                                                        class="h-3 w-3"
                                                        fill="currentColor"
                                                        viewBox="0 0 20 20"
                                                    >
                                                        <path
                                                            fill-rule="evenodd"
                                                            d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
                                                            clip-rule="evenodd"
                                                        />
                                                    </svg>
                                                    Reactivated
                                                    <button
                                                        onclick={(e) => {
                                                            e.stopPropagation();
                                                            dismissReactivation(
                                                                server.id,
                                                            );
                                                        }}
                                                        class="ml-0.5 text-primary hover:text-primary/80"
                                                        title="Dismiss"
                                                    >
                                                        <svg
                                                            class="h-3 w-3"
                                                            fill="currentColor"
                                                            viewBox="0 0 20 20"
                                                        >
                                                            <path
                                                                fill-rule="evenodd"
                                                                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                                                                clip-rule="evenodd"
                                                            />
                                                        </svg>
                                                    </button>
                                                </span>
                                            {/if}
                                        </div>
                                    </td>
                                    <td
                                        class="px-4 py-3.5 text-sm text-foreground"
                                    >
                                        {server.ip_address_v4 ||
                                            server.configured_ip ||
                                            "-"}
                                    </td>
                                    <td
                                        class="px-4 py-3.5 text-right text-sm text-muted-foreground"
                                    >
                                        {formatRelativeTime(server.last_seen)}
                                    </td>
                                    <td class="px-4 py-3.5 text-right">
                                        <div
                                            class="flex items-center justify-end gap-3"
                                        >
                                            <button
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    goto(
                                                        `/servers/${server.id}`,
                                                    );
                                                }}
                                                class="text-sm font-medium text-primary hover:text-primary/80 transition-colors"
                                            >
                                                View
                                            </button>
                                            <button
                                                onclick={(e) =>
                                                    openDeleteModal(server, e)}
                                                class="text-sm font-medium text-destructive hover:text-destructive/80 transition-colors"
                                            >
                                                Delete
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>

                <!-- Pagination -->
                {#if totalPages > 1}
                    <div
                        class="flex items-center justify-between border-t px-4 py-3"
                    >
                        <p class="text-sm text-muted-foreground">
                            {(page - 1) * PER_PAGE + 1}-{Math.min(
                                page * PER_PAGE,
                                total,
                            )} of {total} servers
                        </p>
                        <div class="flex items-center gap-2">
                            <button
                                onclick={() => loadPage(page - 1)}
                                disabled={page <= 1}
                                class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Previous
                            </button>
                            <span class="text-sm text-muted-foreground">
                                {page} / {totalPages}
                            </span>
                            <button
                                onclick={() => loadPage(page + 1)}
                                disabled={page >= totalPages}
                                class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Next
                            </button>
                        </div>
                    </div>
                {/if}
            </div>
        {/if}
    </main>
</div>

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        role="presentation"
        onclick={cancelDelete}
    >
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div
            class="w-full max-w-md rounded-lg border bg-card p-4 sm:p-6 shadow-lg mx-4 sm:mx-0"
            role="dialog"
            aria-modal="true"
            tabindex="-1"
            onclick={(e) => e.stopPropagation()}
        >
            <h3 class="text-lg font-semibold text-foreground mb-3">
                Confirm Delete
            </h3>
            <p class="text-sm text-muted-foreground mb-4">
                Are you sure you want to delete the server "{serverToDelete?.name}"?
            </p>
            {#if serverToDelete?.status === "online"}
                <div
                    class="mb-4 rounded-md border border-primary/20 bg-primary/5 p-3"
                >
                    <p class="text-sm text-foreground">
                        Note: This will remove the server from the database, but
                        the agent will remain installed on the server. You will
                        need to uninstall it manually.
                    </p>
                </div>
            {/if}
            <p class="text-sm font-medium text-destructive mb-6">
                This action cannot be undone.
            </p>
            <div class="flex gap-3 justify-end">
                <button
                    onclick={cancelDelete}
                    class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
                >
                    Cancel
                </button>
                <button
                    onclick={handleDelete}
                    class="rounded-lg bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90"
                >
                    Delete Server
                </button>
            </div>
        </div>
    </div>
{/if}
