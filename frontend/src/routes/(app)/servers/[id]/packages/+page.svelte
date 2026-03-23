<script lang="ts">
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
    import { PACKAGES_PER_PAGE, COLLECTIONS_PER_PAGE } from "$lib/constants";
    import type {
        Server,
        Package,
        PackageStats,
        PackageCollection,
        PackageHistory,
    } from "$lib/types";
    import Pagination from "$lib/components/Pagination.svelte";
    import {
        getManagerLabel,
        getManagerColor,
        formatDateTime,
    } from "$lib/utils";
    import { userStore } from "$lib/stores/user";
    import { Filter, ChevronDown } from "lucide-svelte";

    const timeFormat = $derived(($userStore.user?.time_format ?? '24h') as '12h' | '24h');

    let server: Server | null = $state(null);
    let packages: Package[] = $state([]);
    let stats: PackageStats | null = $state(null);
    let collections: PackageCollection[] = $state([]);
    let history: PackageHistory[] = $state([]);
    let loading = $state(true);
    let error = $state("");
    let searchTerm = $state("");
    let selectedManagers: Set<string> = $state(new Set());
    let allManagerKeys: string[] = $state([]);
    let totalCount = $state(0);
    let limit = PACKAGES_PER_PAGE;
    let offset = $state(0);
    let showCollections = $state(false);
    let showHistory = $state(false);

    let serverId = $derived($page.params.id);
    // Filtered = at least one manager is hidden
    let isFiltered = $derived(
        allManagerKeys.length > 0 &&
            selectedManagers.size < allManagerKeys.length,
    );
    let filterLabel = $derived(
        !isFiltered
            ? "All packages"
            : selectedManagers.size === 1
              ? getManagerLabel([...selectedManagers][0])
              : `${selectedManagers.size} packages`,
    );
    let currentPage = $derived(Math.floor(offset / limit) + 1);
    let totalPages = $derived(Math.ceil(totalCount / limit));

    onMount(async () => {
        await loadData();
    });

    // Full load: server info, stats, collections + packages
    async function loadData() {
        loading = true;
        try {
            const [serverData, packagesData, statsData, collectionsData] =
                await Promise.all([
                    api.getServer(serverId),
                    api.getServerPackages(serverId, {
                        limit,
                        offset,
                        package_manager: isFiltered
                            ? [...selectedManagers].join(",")
                            : undefined,
                        search: searchTerm || undefined,
                    }),
                    api.getPackageStats(serverId),
                    api.getPackageCollections(serverId, {
                        limit: COLLECTIONS_PER_PAGE,
                    }),
                ]);

            server = serverData.server;
            packages = packagesData.packages || [];
            totalCount = packagesData.total_count || 0;
            stats = statsData;
            collections = collectionsData.collections || [];

            // On first load, initialise selectedManagers with all available managers
            if (
                allManagerKeys.length === 0 &&
                statsData.by_package_manager?.length > 0
            ) {
                allManagerKeys = statsData.by_package_manager.map(
                    (pm) => pm.package_manager,
                );
                selectedManagers = new Set(allManagerKeys);
            }
        } catch (err: unknown) {
            error =
                err instanceof Error ? err.message : "Failed to load packages";
        } finally {
            loading = false;
        }
    }

    // Partial reload: packages only (search, filter, pagination)
    let tableLoading = $state(false);

    async function loadPackages() {
        tableLoading = true;
        try {
            const packagesData = await api.getServerPackages(serverId, {
                limit,
                offset,
                package_manager: isFiltered
                    ? [...selectedManagers].join(",")
                    : undefined,
                search: searchTerm || undefined,
            });
            packages = packagesData.packages || [];
            totalCount = packagesData.total_count || 0;
        } catch (err: unknown) {
            error =
                err instanceof Error ? err.message : "Failed to load packages";
        } finally {
            tableLoading = false;
        }
    }

    async function loadHistory() {
        try {
            const data = await api.getPackageHistory(serverId, {
                limit: 50,
                exclude_initial: true,
            });
            history = data.history || [];
        } catch {
            // silently ignore — history is non-critical
        }
    }

    let searchDebounce: ReturnType<typeof setTimeout>;

    function handleSearchInput() {
        clearTimeout(searchDebounce);
        searchDebounce = setTimeout(() => {
            offset = 0;
            loadPackages();
        }, 300);
    }

    async function handleSearch() {
        clearTimeout(searchDebounce);
        offset = 0;
        await loadPackages();
    }

    function toggleManager(manager: string) {
        const next = new Set(selectedManagers);
        if (next.has(manager)) {
            next.delete(manager);
            // Don't allow deselecting the last one — reset to all instead
            if (next.size === 0) {
                selectedManagers = new Set(allManagerKeys);
                offset = 0;
                loadPackages();
                return;
            }
        } else {
            next.add(manager);
        }
        selectedManagers = next;
        offset = 0;
        loadPackages();
    }

    function clearFilter() {
        selectedManagers = new Set(allManagerKeys);
        offset = 0;
        loadPackages();
    }

    function handlePageChange(page: number) {
        offset = (page - 1) * limit;
        loadPackages();
    }

    async function toggleHistory() {
        showHistory = !showHistory;
        if (showHistory && history.length === 0) {
            await loadHistory();
        }
    }

    function formatDuration(ms: number): string {
        if (!ms) return "-";
        if (ms < 1000) return `${ms}ms`;
        return `${(ms / 1000).toFixed(1)}s`;
    }

    const CHANGE_TYPE_STYLES: Record<string, string> = {
        added: "bg-green-500/10 text-green-600 border-green-500/20 dark:text-green-400",
        removed: "bg-destructive/10 text-destructive border-destructive/20",
        updated: "bg-(--chart-4)/10 text-(--chart-4) border-(--chart-4)/20",
    };

    function getChangeTypeStyle(changeType: string): string {
        return (
            CHANGE_TYPE_STYLES[changeType] ||
            "bg-muted text-muted-foreground border-border"
        );
    }
</script>

<svelte:head>
    <title>Packages{server ? ` - ${server.name}` : ""} - Watchflare</title>
</svelte:head>

<!-- Back Link -->
<div class="mb-6">
    <a
        href="/servers/{serverId}"
        class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
    >
        <svg
            class="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M15 19l-7-7 7-7"
            />
        </svg>
        Back to Server
    </a>
</div>

<!-- Header -->
<div class="mb-6">
    <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground mb-1">Package Inventory</p>
    <h1 class="text-lg font-semibold text-foreground">{server?.name ?? '…'}</h1>
    {#if server}
        <p class="text-sm text-muted-foreground mt-1 flex items-center flex-wrap gap-x-3">
            {#if server.hostname}
                <span>{server.hostname}</span>
            {/if}
            {#if server.ip_address_v4 || server.configured_ip}
                {#if server.hostname}<span class="text-border">|</span>{/if}
                <span>{server.ip_address_v4 || server.configured_ip}</span>
            {/if}
        </p>
    {/if}
</div>

<!-- Error -->
{#if error}
    <div
        class="mb-6 rounded-lg border border-destructive bg-destructive/10 p-4"
    >
        <p class="text-sm text-destructive">{error}</p>
    </div>
{/if}

<!-- Stats -->
{#if stats}
    {@const managerCards = (stats.by_package_manager || []).slice(0, 2)}
    <div class="flex flex-wrap gap-3 mb-6">
        <div class="rounded-lg border bg-card p-3 w-40">
            <p class="text-xs text-muted-foreground mb-1">Total Packages</p>
            <p class="text-lg font-semibold text-foreground">{stats.total_packages || 0}</p>
        </div>
        <div class="rounded-lg border bg-card p-3 w-40">
            <p class="text-xs text-muted-foreground mb-1">Recent Changes (30d)</p>
            <p class="text-lg font-semibold text-foreground">{stats.recent_changes || 0}</p>
        </div>
        {#each managerCards as pm}
            <div class="rounded-lg border bg-card p-3 w-40">
                <p class="text-xs text-muted-foreground mb-1">{getManagerLabel(pm.package_manager)}</p>
                <p class="text-lg font-semibold text-foreground">{pm.count}</p>
            </div>
        {/each}
    </div>
{/if}

<!-- Search & Filters -->
<div class="mb-6 rounded-lg border bg-card p-3">
    <div class="flex flex-col sm:flex-row gap-3">
        <div class="flex-1">
            <input
                type="text"
                bind:value={searchTerm}
                oninput={handleSearchInput}
                onkeydown={(e) => e.key === "Enter" && handleSearch()}
                placeholder="Search packages..."
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
        </div>
        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        class="w-fit inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-colors whitespace-nowrap
							{isFiltered
                            ? 'border-primary bg-primary/5 text-primary hover:bg-primary/10'
                            : 'bg-background text-foreground hover:bg-muted'}"
                    >
                        <Filter class="h-4 w-4" />
                        {filterLabel}
                        {#if isFiltered}
                            <span
                                class="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-primary px-1 text-xs font-medium text-primary-foreground"
                            >
                                {selectedManagers.size}
                            </span>
                        {/if}
                        <ChevronDown class="h-3.5 w-3.5 opacity-50" />
                    </button>
                {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end">
                {#each [...(stats?.by_package_manager || [])].sort((a, b) => b.count - a.count) as pm}
                    <DropdownMenu.Item
                        closeOnSelect={false}
                        onclick={() => toggleManager(pm.package_manager)}
                    >
                        <div
                            class="flex h-4 w-4 shrink-0 items-center justify-center rounded border
							{selectedManagers.has(pm.package_manager)
                                ? 'border-primary bg-primary'
                                : 'border-muted-foreground/40'}"
                        >
                            {#if selectedManagers.has(pm.package_manager)}
                                <svg
                                    class="h-3 w-3 text-primary-foreground"
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                >
                                    <path
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="3"
                                        d="M5 13l4 4L19 7"
                                    />
                                </svg>
                            {/if}
                        </div>
                        <span class="flex-1"
                            >{getManagerLabel(pm.package_manager)}</span
                        >
                        <span
                            class="ml-4 tabular-nums text-xs text-muted-foreground"
                            >{pm.count}</span
                        >
                    </DropdownMenu.Item>
                {/each}
                {#if isFiltered}
                    <DropdownMenu.Separator />
                    <DropdownMenu.Item
                        onclick={clearFilter}
                        class="text-muted-foreground"
                    >
                        Clear filter
                    </DropdownMenu.Item>
                {/if}
            </DropdownMenu.Content>
        </DropdownMenu.Root>
        <button
            onclick={handleSearch}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
            Search
        </button>
    </div>
</div>

<!-- Packages Table -->
<div class="rounded-lg border bg-card overflow-hidden mb-6">
    {#if loading}
        <div class="flex items-center justify-center py-20">
            <p class="text-muted-foreground">Loading packages...</p>
        </div>
    {:else if packages.length === 0 && !tableLoading}
        <div
            class="flex flex-col items-center justify-center py-20 text-center"
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
                    d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
                />
            </svg>
            <p class="text-sm text-muted-foreground">No packages found</p>
        </div>
    {:else}
        <div class="overflow-x-auto transition-opacity {tableLoading ? 'opacity-50' : ''}">
            <table class="w-full">
                <thead>
                    <tr class="border-b bg-muted/30">
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                            >Name</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                            >Version</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                            >Manager</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase hidden lg:table-cell"
                            >Architecture</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase hidden xl:table-cell"
                            >Description</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase hidden md:table-cell"
                            >First Seen</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase"
                            >Last Seen</th
                        >
                    </tr>
                </thead>
                <tbody class="divide-y divide-border">
                    {#each packages as pkg}
                        <tr class="hover:bg-muted/20 transition-colors">
                            <td
                                class="px-4 py-3 text-sm font-medium text-foreground"
                            >
                                {pkg.name}
                            </td>
                            <td
                                class="px-4 py-3 text-sm font-mono text-muted-foreground"
                            >
                                {pkg.version || "-"}
                            </td>
                            <td class="px-4 py-3">
                                <span
                                    class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getManagerColor(
                                        pkg.package_manager,
                                    )}"
                                >
                                    {getManagerLabel(pkg.package_manager)}
                                </span>
                            </td>
                            <td
                                class="px-4 py-3 text-sm text-muted-foreground hidden lg:table-cell"
                            >
                                {pkg.architecture || "-"}
                            </td>
                            <td
                                class="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate hidden xl:table-cell"
                            >
                                {pkg.description || "-"}
                            </td>
                            <td
                                class="px-4 py-3 text-right text-sm text-muted-foreground hidden md:table-cell"
                            >
                                {formatDateTime(pkg.first_seen, timeFormat)}
                            </td>
                            <td
                                class="px-4 py-3 text-right text-sm text-muted-foreground"
                            >
                                {formatDateTime(pkg.last_seen, timeFormat)}
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>

        <!-- Pagination -->
        <Pagination
            {currentPage}
            {totalPages}
            totalItems={totalCount}
            pageSize={limit}
            itemLabel="packages"
            onPageChange={handlePageChange}
        />
    {/if}
</div>

<!-- Recent Changes -->
<div class="mb-4">
    <button
        onclick={toggleHistory}
        class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
    >
        <svg
            class="h-4 w-4 transition-transform {showHistory
                ? 'rotate-90'
                : ''}"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 5l7 7-7 7"
            />
        </svg>
        {showHistory ? "Hide" : "Show"} Recent Changes
    </button>

    {#if showHistory}
        <div class="mt-4 rounded-lg border bg-card overflow-hidden">
            {#if history.length === 0}
                <div class="flex items-center justify-center py-10">
                    <p class="text-sm text-muted-foreground">
                        No package changes recorded yet
                    </p>
                </div>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead>
                            <tr class="border-b bg-muted/30">
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                                    >Date</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                                    >Change</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                                    >Package</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                                    >Version</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase hidden sm:table-cell"
                                    >Manager</th
                                >
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-border">
                            {#each history as entry}
                                <tr class="hover:bg-muted/20 transition-colors">
                                    <td
                                        class="px-4 py-3 text-sm text-muted-foreground whitespace-nowrap"
                                    >
                                        {formatDateTime(entry.timestamp, timeFormat)}
                                    </td>
                                    <td class="px-4 py-3">
                                        <span
                                            class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getChangeTypeStyle(
                                                entry.change_type,
                                            )}"
                                        >
                                            {entry.change_type}
                                        </span>
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm font-medium text-foreground"
                                    >
                                        {entry.name}
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm font-mono text-muted-foreground"
                                    >
                                        {entry.version || "-"}
                                    </td>
                                    <td class="px-4 py-3 hidden sm:table-cell">
                                        <span
                                            class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getManagerColor(
                                                entry.package_manager,
                                            )}"
                                        >
                                            {getManagerLabel(
                                                entry.package_manager,
                                            )}
                                        </span>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {/if}
</div>

<!-- Collection History -->
<div class="mb-6">
    <button
        onclick={() => (showCollections = !showCollections)}
        class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
    >
        <svg
            class="h-4 w-4 transition-transform {showCollections
                ? 'rotate-90'
                : ''}"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 5l7 7-7 7"
            />
        </svg>
        {showCollections ? "Hide" : "Show"} Collection History
    </button>

    {#if showCollections && collections.length > 0}
        <div class="mt-4 rounded-lg border bg-card overflow-hidden">
            <table class="w-full">
                <thead>
                    <tr class="border-b bg-muted/30">
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                            >Date</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase"
                            >Type</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase"
                            >Packages</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase hidden sm:table-cell"
                            >Changes</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase hidden md:table-cell"
                            >Duration</th
                        >
                    </tr>
                </thead>
                <tbody class="divide-y divide-border">
                    {#each collections as collection}
                        <tr class="hover:bg-muted/20">
                            <td
                                class="px-4 py-3 text-sm text-foreground whitespace-nowrap"
                            >
                                {formatDateTime(collection.timestamp, timeFormat)}
                            </td>
                            <td class="px-4 py-3">
                                <span
                                    class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {collection.collection_type ===
                                        'full' ||
                                    collection.collection_type === 'initial'
                                        ? 'bg-primary/10 text-primary border-primary/20'
                                        : 'bg-muted text-muted-foreground border-border'}"
                                >
                                    {collection.collection_type}
                                </span>
                            </td>
                            <td
                                class="px-4 py-3 text-right text-sm text-foreground"
                            >
                                {collection.package_count}
                            </td>
                            <td
                                class="px-4 py-3 text-right text-sm text-foreground hidden sm:table-cell"
                            >
                                {collection.changes_count || 0}
                            </td>
                            <td
                                class="px-4 py-3 text-right text-sm text-muted-foreground hidden md:table-cell"
                            >
                                {formatDuration(collection.duration_ms)}
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>
