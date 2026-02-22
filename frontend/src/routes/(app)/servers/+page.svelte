<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { goto } from "$app/navigation";
    import * as api from "$lib/api.js";
    import { handleSSEReactivation, logger } from "$lib/utils";
    import type { Server, SSEEvent } from "$lib/types";
    import { sseStore } from "$lib/stores/sse";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Pagination from "$lib/components/Pagination.svelte";
    import ServerFilters from "$lib/components/server/ServerFilters.svelte";
    import ServerListTable from "$lib/components/server/ServerListTable.svelte";

    const PER_PAGE = 20;

    let servers: Server[] = $state([]);
    let total = $state(0);
    let page = $state(1);
    let initialLoading = $state(true);
    let loading = $state(false);
    let error = $state("");
    let showDeleteConfirm = $state(false);
    let serverToDelete: Server | null = $state(null);
    let sseUnsubscribe: (() => void) | null = null;

    // Sort state
    let sortColumn = $state("created_at");
    let sortOrder: "asc" | "desc" = $state("desc");

    // Filter state
    let searchQuery = $state("");
    let statusFilter = $state("");
    let environmentFilter = $state("");
    let searchTimeout: ReturnType<typeof setTimeout> | null = null;

    let totalPages = $derived(Math.max(1, Math.ceil(total / PER_PAGE)));

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
            error =
                err instanceof Error ? err.message : "Failed to load servers";
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

    function handleStatusChange(value: string) {
        statusFilter = value;
        loadPage(1);
    }

    function handleEnvironmentChange(value: string) {
        environmentFilter = value;
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

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);

        if (event.type === "server_update") {
            const update = event.data;

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
            error =
                err instanceof Error ? err.message : "Failed to delete server";
            showDeleteConfirm = false;
            serverToDelete = null;
        }
    }
</script>

<svelte:head>
    <title>Servers - Watchflare</title>
</svelte:head>

<!-- Header -->
<div class="mb-6">
    <h1 class="text-xl sm:text-2xl font-semibold text-foreground">Servers</h1>
    <p class="text-sm text-muted-foreground mt-1">
        Manage your monitored servers
    </p>
</div>

<!-- Filters -->
<ServerFilters
    {searchQuery}
    {statusFilter}
    {environmentFilter}
    onSearchInput={handleSearchInput}
    onStatusChange={handleStatusChange}
    onEnvironmentChange={handleEnvironmentChange}
/>

{#if initialLoading}
    <div class="flex items-center justify-center py-20">
        <p class="text-muted-foreground">Loading servers...</p>
    </div>
{:else if error}
    <div class="rounded-lg border border-destructive bg-destructive/10 p-4">
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
        <p class="text-sm text-muted-foreground">
            No servers match your filters
        </p>
    </div>
{:else}
    <div class="rounded-lg border bg-card overflow-x-auto">
        <ServerListTable
            {servers}
            {sortColumn}
            {sortOrder}
            onSort={handleSort}
            onDelete={openDeleteModal}
            onDismissReactivation={dismissReactivation}
        />

        <!-- Pagination -->
        <Pagination currentPage={page} {totalPages} totalItems={total} pageSize={PER_PAGE} itemLabel="servers" onPageChange={loadPage} />
    </div>
{/if}
<!-- Delete Confirmation Modal -->
<ConfirmDialog
    open={showDeleteConfirm}
    title="Confirm Delete"
    onConfirm={handleDelete}
    onClose={cancelDelete}
    confirmLabel="Delete Server"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">
        Are you sure you want to delete the server "{serverToDelete?.name}"?
    </p>
    {#if serverToDelete?.status === "online"}
        <div class="mb-4 rounded-md border border-primary/20 bg-primary/5 p-3">
            <p class="text-sm text-foreground">
                Note: This will remove the server from the database, but
                the agent will remain installed on the server. You will
                need to uninstall it manually.
            </p>
        </div>
    {/if}
    <p class="text-sm font-medium text-destructive">
        This action cannot be undone.
    </p>
</ConfirmDialog>
