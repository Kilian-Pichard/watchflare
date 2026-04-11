<script lang="ts">
    import { onMount, getContext } from "svelte";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
    import { PACKAGES_PER_PAGE, COLLECTIONS_PER_PAGE } from "$lib/constants";
    import type {
        Host,
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
    import { Filter, ChevronDown, ChevronRight } from "lucide-svelte";

    const timeFormat = $derived(($userStore.user?.time_format ?? '24h') as '12h' | '24h');

    type PackagesCache = {
        packages: Package[]; stats: PackageStats | null; collections: PackageCollection[];
        totalCount: number; offset: number; searchTerm: string;
        allManagerKeys: string[]; selectedManagers: string[];
    };
    const ctx = getContext<{
        host: Host | null;
        packagesCache: PackagesCache | null;
        setPackagesCache: (data: PackagesCache) => void;
    }>('hostDetail');

    const cached = ctx.packagesCache;
    let packages: Package[] = $state(cached?.packages ?? []);
    let stats: PackageStats | null = $state(cached?.stats ?? null);
    let collections: PackageCollection[] = $state(cached?.collections ?? []);
    let history: PackageHistory[] = $state([]);
    let loading = $state(!cached);
    let error = $state("");
    let searchTerm = $state(cached?.searchTerm ?? "");
    let selectedManagers: Set<string> = $state(new Set(cached?.selectedManagers ?? []));
    let allManagerKeys: string[] = $state(cached?.allManagerKeys ?? []);
    let totalCount = $state(cached?.totalCount ?? 0);
    let limit = PACKAGES_PER_PAGE;
    let offset = $state(cached?.offset ?? 0);
    let showCollections = $state(false);
    let showHistory = $state(false);

    let hostId = $derived($page.params.id);
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
              : `${selectedManagers.size} managers`,
    );
    let currentPage = $derived(Math.floor(offset / limit) + 1);
    let totalPages = $derived(Math.ceil(totalCount / limit));

    onMount(async () => {
        await loadData(!!cached);
    });

    function saveToCache() {
        ctx.setPackagesCache({
            packages, stats, collections, totalCount, offset, searchTerm,
            allManagerKeys, selectedManagers: [...selectedManagers],
        });
    }

    // Full load: stats, collections + packages
    // silent=true → no loading spinner, data updates quietly (used on tab return)
    async function loadData(silent = false) {
        if (!silent) loading = true;
        try {
            const [packagesData, statsData, collectionsData] =
                await Promise.all([
                    api.getHostPackages(hostId, {
                        limit,
                        offset,
                        package_manager: isFiltered
                            ? [...selectedManagers].join(",")
                            : undefined,
                        search: searchTerm || undefined,
                    }),
                    api.getPackageStats(hostId),
                    api.getPackageCollections(hostId, {
                        limit: COLLECTIONS_PER_PAGE,
                    }),
                ]);

            packages = packagesData.packages || [];
            totalCount = packagesData.total_count || 0;
            stats = statsData;
            collections = collectionsData.collections || [];

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
            if (!silent) error = err instanceof Error ? err.message : "Failed to load packages";
        } finally {
            if (!silent) loading = false;
            saveToCache();
        }
    }

    // Partial reload: packages only (search, filter, pagination)
    let tableLoading = $state(false);

    async function loadPackages() {
        tableLoading = true;
        try {
            const packagesData = await api.getHostPackages(hostId, {
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
            saveToCache();
        }
    }

    async function loadHistory() {
        try {
            const data = await api.getPackageHistory(hostId, {
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
        offset = 0;
        clearTimeout(searchDebounce);
        searchDebounce = setTimeout(() => {
            loadPackages();
        }, 300);
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
    <title>Packages{ctx.host ? ` - ${ctx.host.display_name}` : ''} - Watchflare</title>
</svelte:head>

<!-- Error -->
{#if error}
    <div
        class="mb-6 rounded-lg border border-destructive bg-destructive/10 p-4"
    >
        <p class="text-sm text-destructive">{error}</p>
    </div>
{/if}

<!-- Stats -->
{#if loading}
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-4">
        {#each Array(4) as _}
            <div class="rounded-xl border bg-card px-4 py-3.5 animate-pulse">
                <div class="h-3 w-20 rounded bg-muted mb-3"></div>
                <div class="h-7 w-12 rounded bg-muted"></div>
            </div>
        {/each}
    </div>
{:else if stats}
    {@const managerCards = (stats.by_package_manager || []).slice(0, 2)}
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-4">
        <div class="rounded-xl border bg-card px-4 py-3.5">
            <p class="text-xs text-muted-foreground mb-1.5">Total packages</p>
            <p class="text-2xl font-semibold tabular-nums text-foreground">{stats.total_packages || 0}</p>
        </div>
        <div class="rounded-xl border bg-card px-4 py-3.5">
            <p class="text-xs text-muted-foreground mb-1.5">Changes (30d)</p>
            <p class="text-2xl font-semibold tabular-nums text-foreground">{stats.recent_changes || 0}</p>
        </div>
        {#each managerCards as pm}
            <div class="rounded-xl border bg-card px-4 py-3.5">
                <p class="text-xs text-muted-foreground mb-1.5 truncate">{getManagerLabel(pm.package_manager)}</p>
                <p class="text-2xl font-semibold tabular-nums text-foreground">{pm.count}</p>
            </div>
        {/each}
    </div>
{/if}

<!-- Search & Filters -->
<div class="mb-4 flex items-center gap-2">
    <input
        type="text"
        bind:value={searchTerm}
        oninput={handleSearchInput}
        onkeydown={(e) => { if (e.key === 'Enter') { clearTimeout(searchDebounce); offset = 0; loadPackages(); } }}
        placeholder="Search packages..."
        class="flex-1 rounded-lg border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
    />
    <DropdownMenu.Root>
        <DropdownMenu.Trigger>
            {#snippet child({ props })}
                <button
                    {...props}
                    class="inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors whitespace-nowrap
                        {isFiltered
                        ? 'border-primary/40 bg-primary/5 text-primary hover:bg-primary/10'
                        : 'bg-card text-muted-foreground hover:bg-muted hover:text-foreground'}"
                >
                    <Filter class="h-3.5 w-3.5" />
                    {filterLabel}
                    {#if isFiltered}
                        <span class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-primary/15 px-1 text-xs font-medium text-primary">
                            {selectedManagers.size}
                        </span>
                    {/if}
                    <ChevronDown class="h-3 w-3 opacity-40" />
                </button>
            {/snippet}
        </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end">
                {#each [...(stats?.by_package_manager || [])].sort((a, b) => b.count - a.count) as pm}
                    <DropdownMenu.Item
                        closeOnSelect={false}
                        onclick={() => toggleManager(pm.package_manager)}
                        disabled={selectedManagers.size === 1 && selectedManagers.has(pm.package_manager)}
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
</div>

<!-- Packages Table -->
<div class="rounded-xl border bg-card overflow-hidden mb-6">
    {#if loading}
        <div class="flex items-center justify-center py-20">
            <p class="text-muted-foreground">Loading packages...</p>
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full">
                <thead>
                    <tr class="border-b bg-muted/30">
                        <th scope="col" class="px-4 py-2.5 text-left text-sm font-semibold text-muted-foreground">Name</th>
                        <th scope="col" class="px-4 py-2.5 text-left text-sm font-semibold text-muted-foreground">Version</th>
                        <th scope="col" class="px-4 py-2.5 text-left text-sm font-semibold text-muted-foreground">Manager</th>
                        <th scope="col" class="px-4 py-2.5 text-left text-sm font-semibold text-muted-foreground hidden lg:table-cell">Architecture</th>
                        <th scope="col" class="px-4 py-2.5 text-left text-sm font-semibold text-muted-foreground hidden xl:table-cell">Description</th>
                        <th scope="col" class="px-4 py-2.5 text-right text-sm font-semibold text-muted-foreground hidden md:table-cell">First Seen</th>
                        <th scope="col" class="px-4 py-2.5 text-right text-sm font-semibold text-muted-foreground">Last Seen</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-border transition-opacity {tableLoading ? 'opacity-50 pointer-events-none' : ''}">
                    {#each packages as pkg}
                        <tr class="hover:bg-muted/20 transition-colors">
                            <td class="px-4 py-3 text-sm font-medium text-foreground">{pkg.name}</td>
                            <td class="px-4 py-3 text-sm font-mono text-muted-foreground">{pkg.version || "-"}</td>
                            <td class="px-4 py-3">
                                <span class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getManagerColor(pkg.package_manager)}">
                                    {getManagerLabel(pkg.package_manager)}
                                </span>
                            </td>
                            <td class="px-4 py-3 text-sm text-muted-foreground hidden lg:table-cell">{pkg.architecture || "-"}</td>
                            <td class="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate hidden xl:table-cell">{pkg.description || "-"}</td>
                            <td class="px-4 py-3 text-right text-sm text-muted-foreground hidden md:table-cell">{formatDateTime(pkg.first_seen, timeFormat)}</td>
                            <td class="px-4 py-3 text-right text-sm text-muted-foreground">{formatDateTime(pkg.last_seen, timeFormat)}</td>
                        </tr>
                    {:else}
                        {#if !tableLoading}
                            <tr>
                                <td colspan="7" class="py-16 text-center">
                                    <svg class="mx-auto h-10 w-10 text-muted-foreground/40 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                                    </svg>
                                    <p class="text-sm text-muted-foreground">No packages found</p>
                                </td>
                            </tr>
                        {/if}
                    {/each}
                </tbody>
            </table>
        </div>

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
        <ChevronRight class="h-4 w-4 transition-transform {showHistory ? 'rotate-90' : ''}" />
        {showHistory ? "Hide" : "Show"} Recent Changes
    </button>

    {#if showHistory}
        <div class="mt-4 rounded-xl border bg-card overflow-hidden">
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
                                    class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground"
                                    >Date</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground"
                                    >Change</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground"
                                    >Package</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground"
                                    >Version</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground hidden sm:table-cell"
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
        <ChevronRight class="h-4 w-4 transition-transform {showCollections ? 'rotate-90' : ''}" />
        {showCollections ? "Hide" : "Show"} Collection History
    </button>

    {#if showCollections && collections.length > 0}
        <div class="mt-4 rounded-xl border bg-card overflow-hidden">
            <table class="w-full">
                <thead>
                    <tr class="border-b bg-muted/30">
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground"
                            >Date</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-sm font-semibold text-muted-foreground"
                            >Type</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-sm font-semibold text-muted-foreground"
                            >Packages</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-sm font-semibold text-muted-foreground hidden sm:table-cell"
                            >Changes</th
                        >
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-sm font-semibold text-muted-foreground hidden md:table-cell"
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
