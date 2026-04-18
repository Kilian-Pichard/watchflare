<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import { handleSSEReactivation, logger } from "$lib/utils";
    import { HOSTS_PER_PAGE, SEARCH_DEBOUNCE_MS } from "$lib/constants";
    import type { Host, SSEEvent } from "$lib/types";
    import { sseStore } from "$lib/stores/sse";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Pagination from "$lib/components/Pagination.svelte";
    import HostFilters from "$lib/components/host/HostFilters.svelte";
    import HostListTable from "$lib/components/host/HostListTable.svelte";
    import AddHostModal from "$lib/components/host/AddHostModal.svelte";
    import DataTable from "$lib/components/DataTable.svelte";

    const PER_PAGE = HOSTS_PER_PAGE;

    let hosts: Host[] = $state([]);
    let total = $state(0);
    let currentPage = $state(1);
    let initialLoading = $state(true);
    let loading = $state(false);
    let latestAgentVersion: string | null = $state(null);
    let error = $state("");
    let showAddHost = $state(false);

    $effect(() => {
        if (($page.state as { openAddHost?: boolean }).openAddHost) {
            showAddHost = true;
        }
    });
    let showDeleteConfirm = $state(false);
    let hostToDelete: Host | null = $state(null);
    let sseUnsubscribe: (() => void) | null = null;

    // Sort state
    let sortColumn = $state("created_at");
    let sortOrder: "asc" | "desc" = $state("desc");

    // Filter state
    let searchQuery = $state("");
    let statusFilter = $state("");
    let searchTimeout: ReturnType<typeof setTimeout> | null = null;

    let totalPages = $derived(Math.max(1, Math.ceil(total / PER_PAGE)));

    async function loadPage(p: number) {
        loading = true;
        error = "";
        try {
            const [response, versionResponse] = await Promise.all([
                api.listHosts({
                    page: p,
                    perPage: PER_PAGE,
                    sort: sortColumn,
                    order: sortOrder,
                    status: statusFilter || undefined,
                    search: searchQuery || undefined,
                }),
                latestAgentVersion === null ? api.getLatestAgentVersion().catch(() => ({ latest_version: '' })) : Promise.resolve({ latest_version: latestAgentVersion }),
            ]);
            hosts = response.hosts || [];
            total = response.total || 0;
            currentPage = p;
            if (latestAgentVersion === null) latestAgentVersion = versionResponse.latest_version || null;
        } catch (err: unknown) {
            error =
                err instanceof Error ? err.message : "Failed to load hosts";
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
            sortOrder = "desc";
        }
        loadPage(currentPage);
    }

    function handleSearchInput(e: Event) {
        const value = (e.target as HTMLInputElement).value;
        searchQuery = value;
        if (searchTimeout) clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            loadPage(1);
        }, SEARCH_DEBOUNCE_MS);
    }

    function handleStatusChange(value: string) {
        statusFilter = value;
        loadPage(1);
    }

    async function handleDismissReactivation(hostId: string) {
        try {
            await api.dismissReactivation(hostId);
            await loadPage(currentPage);
        } catch (err) {
            logger.error("Failed to dismiss reactivation:", err);
        }
    }

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);

        if (event.type === "host_update") {
            const update = event.data;

            const hostIndex = hosts.findIndex((s) => s.id === update.id);
            if (hostIndex !== -1) {
                hosts[hostIndex] = {
                    ...hosts[hostIndex],
                    status: update.status,
                    ip_address_v4: update.ip_address_v4,
                    ip_address_v6: update.ip_address_v6,
                    configured_ip: update.configured_ip,
                    ignore_ip_mismatch: update.ignore_ip_mismatch,
                    last_seen: update.last_seen,
                };
                hosts = [...hosts];
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

    function openDeleteModal(host: Host, e: Event) {
        e.stopPropagation();
        hostToDelete = host;
        showDeleteConfirm = true;
    }

    function cancelDelete() {
        showDeleteConfirm = false;
        hostToDelete = null;
    }

    async function handleDelete() {
        if (!hostToDelete) return;

        try {
            await api.deleteHost(hostToDelete.id);
            await loadPage(currentPage);
            showDeleteConfirm = false;
            hostToDelete = null;
        } catch (err: unknown) {
            error =
                err instanceof Error ? err.message : "Failed to delete host";
            showDeleteConfirm = false;
            hostToDelete = null;
        }
    }
</script>

<svelte:head>
    <title>Hosts - Watchflare</title>
</svelte:head>

<!-- Header -->
<div class="mb-6">
    <h1 class="text-xl sm:text-2xl font-semibold text-foreground">Hosts</h1>
    <p class="text-sm text-muted-foreground mt-1">
        Manage your monitored hosts
    </p>
</div>

<!-- Filters -->
<HostFilters
    {searchQuery}
    {statusFilter}
    onSearchInput={handleSearchInput}
    onStatusChange={handleStatusChange}
/>

{#if initialLoading}
    <div class="flex items-center justify-center py-20">
        <p class="text-muted-foreground">Loading hosts...</p>
    </div>
{:else if error}
    <div class="rounded-lg border border-destructive bg-destructive/10 p-4">
        <p class="text-sm text-destructive">{error}</p>
    </div>
{:else if hosts.length === 0 && currentPage === 1 && !searchQuery && !statusFilter}
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
            No hosts configured yet
        </h3>
        <p class="text-sm text-muted-foreground mb-6">
            Add your first host to start monitoring
        </p>
        <button
            onclick={() => (showAddHost = true)}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
            Add Your First Host
        </button>
    </div>
{:else if hosts.length === 0}
    <div
        class="flex flex-col items-center justify-center rounded-lg border bg-card py-12 text-center"
    >
        <p class="text-sm text-muted-foreground">
            No hosts match your filters
        </p>
    </div>
{:else}
    <DataTable grow>
        <HostListTable
            {hosts}
            {sortColumn}
            {sortOrder}
            {latestAgentVersion}
            onSort={handleSort}
            onDelete={openDeleteModal}
            onDismissReactivation={handleDismissReactivation}
        />
        {#snippet footer()}
            <Pagination currentPage={currentPage} {totalPages} totalItems={total} pageSize={PER_PAGE} itemLabel="hosts" onPageChange={loadPage} />
        {/snippet}
    </DataTable>
{/if}
<AddHostModal open={showAddHost} onClose={() => (showAddHost = false)} />

<!-- Delete Confirmation Modal -->
<ConfirmDialog
    open={showDeleteConfirm}
    title="Confirm Delete"
    onConfirm={handleDelete}
    onClose={cancelDelete}
    confirmLabel="Delete Host"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">
        Are you sure you want to delete the host "{hostToDelete?.display_name}"?
    </p>
    {#if hostToDelete?.status === "online"}
        <div class="mb-4 rounded-md border border-primary/20 bg-primary/5 p-3">
            <p class="text-sm text-foreground">
                Note: This will remove the host from the database, but
                the agent will remain installed on the host. You will
                need to uninstall it manually.
            </p>
        </div>
    {/if}
    <p class="text-sm font-medium text-destructive">
        This action cannot be undone.
    </p>
</ConfirmDialog>
