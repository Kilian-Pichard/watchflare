<script lang="ts">
    import { onMount, onDestroy, getContext } from "svelte";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
    import { PACKAGES_PER_PAGE } from "$lib/constants";
    import type { Host, Package, PackageStats } from "$lib/types";
    import Pagination from "$lib/components/Pagination.svelte";
    import {
        getManagerLabel,
        getManagerColor,
        formatDateTime,
    } from "$lib/utils";
    import { userStore } from "$lib/stores/user";
    import {
        Filter,
        Columns3,
        ChevronDown,
        RefreshCw,
        ShieldAlert,
        ArrowUp,
        Package as PackageIcon,
        Tag,
        X,
    } from "lucide-svelte";

    const timeFormat = $derived(
        ($userStore.user?.time_format ?? "24h") as "12h" | "24h",
    );

    const ALL_COLUMNS = [
        { key: "name", label: "Name", defaultVisible: true },
        { key: "version", label: "Version", defaultVisible: true },
        { key: "status", label: "Status", defaultVisible: true },
        { key: "manager", label: "Manager", defaultVisible: true },
        {
            key: "latest_version",
            label: "Latest Version",
            defaultVisible: true,
        },
        { key: "arch", label: "Architecture", defaultVisible: false },
        { key: "description", label: "Description", defaultVisible: false },
        { key: "first_seen", label: "First Seen", defaultVisible: false },
        { key: "last_seen", label: "Last Seen", defaultVisible: true },
    ] as const;

    type ColumnKey = (typeof ALL_COLUMNS)[number]["key"];

    const DEFAULT_VISIBLE_COLUMNS = ALL_COLUMNS.filter(
        (c) => c.defaultVisible,
    ).map((c) => c.key) as ColumnKey[];

    type PackagesCache = {
        allPackages: Package[];
        stats: PackageStats | null;
        searchTerm: string;
        allManagerKeys: string[];
        selectedManagers: string[];
        visibleColumns: string[];
    };
    const ctx = getContext<{
        host: Host | null;
        packagesCache: PackagesCache | null;
        setPackagesCache: (data: PackagesCache) => void;
        packageInventorySignal: number;
    }>("hostDetail");

    const cached = ctx.packagesCache;
    // allPackages = full list fetched once from API; all filtering/sorting/pagination is client-side
    let allPackages: Package[] = $state(cached?.allPackages ?? []);
    let stats: PackageStats | null = $state(cached?.stats ?? null);
    let loading = $state(!cached);
    let error = $state("");
    let searchTerm = $state(cached?.searchTerm ?? "");
    let selectedManagers: Set<string> = $state(
        new Set(cached?.selectedManagers ?? []),
    );
    let allManagerKeys: string[] = $state(cached?.allManagerKeys ?? []);
    let visibleColumns: Set<ColumnKey> = $state(
        new Set(
            (cached?.visibleColumns ?? DEFAULT_VISIBLE_COLUMNS) as ColumnKey[],
        ),
    );
    let offset = $state(0);
    let quickFilter = $state<"all" | "needs_update" | "outdated" | "security">(
        "all",
    );

    const PAGE_SIZE_OPTIONS = [20, 50, 100, 200];
    let limit = $state(PACKAGES_PER_PAGE);
    const hostId = $derived($page.params.id);

    const col = $derived((key: ColumnKey) => visibleColumns.has(key));

    // Client-side filtering
    const filteredPackages = $derived(() => {
        let result = allPackages;
        if (searchTerm) {
            const q = searchTerm.toLowerCase();
            result = result.filter((p) => p.name.toLowerCase().includes(q));
        }
        if (selectedManagers.size > 0) {
            result = result.filter((p) =>
                selectedManagers.has(p.package_manager),
            );
        }
        if (quickFilter === "needs_update") {
            result = result.filter(
                (p) => p.available_version || p.has_security_update,
            );
        } else if (quickFilter === "outdated") {
            result = result.filter(
                (p) => p.available_version && !p.has_security_update,
            );
        } else if (quickFilter === "security") {
            result = result.filter((p) => p.has_security_update);
        }
        return result;
    });

    // Client-side sorting
    let sortColumn = $state("name");
    let sortOrder = $state<"asc" | "desc">("asc");

    const sortedPackages = $derived(() => {
        return [...filteredPackages()].sort((a, b) => {
            let valA: string;
            let valB: string;
            switch (sortColumn) {
                case "version":
                    valA = a.version || "";
                    valB = b.version || "";
                    break;
                case "manager":
                    valA = a.package_manager || "";
                    valB = b.package_manager || "";
                    break;
                case "status": {
                    const statusOrder = (p: Package) =>
                        p.has_security_update
                            ? "0"
                            : p.available_version
                              ? "1"
                              : "2";
                    valA = statusOrder(a);
                    valB = statusOrder(b);
                    break;
                }
                case "latest_version":
                    valA = a.available_version || "";
                    valB = b.available_version || "";
                    break;
                case "arch":
                    valA = a.architecture || "";
                    valB = b.architecture || "";
                    break;
                case "first_seen":
                    valA = a.first_seen || "";
                    valB = b.first_seen || "";
                    break;
                case "last_seen":
                    valA = a.last_seen || "";
                    valB = b.last_seen || "";
                    break;
                default:
                    valA = a.name || "";
                    valB = b.name || "";
                    break;
            }
            if (valA < valB) return sortOrder === "asc" ? -1 : 1;
            if (valA > valB) return sortOrder === "asc" ? 1 : -1;
            return 0;
        });
    });

    // Client-side pagination
    const paginatedPackages = $derived(() =>
        sortedPackages().slice(offset, offset + limit),
    );
    const totalCount = $derived(filteredPackages().length);
    const currentPage = $derived(Math.floor(offset / limit) + 1);
    const totalPages = $derived(Math.ceil(totalCount / limit));
    const isFiltered = $derived(selectedManagers.size > 0);
    const filterLabel = $derived(
        !isFiltered
            ? "All packages"
            : selectedManagers.size === 1
              ? getManagerLabel([...selectedManagers][0])
              : `${selectedManagers.size} managers`,
    );
    // Badge on Columns button: count of extra columns beyond default
    const extraColumnsCount = $derived(
        [...visibleColumns].filter((k) => !DEFAULT_VISIBLE_COLUMNS.includes(k))
            .length,
    );

    onMount(async () => {
        await loadData(!!cached);
    });

    onDestroy(() => {
        if (searchDebounce) clearTimeout(searchDebounce);
        if (collectErrorTimeout) clearTimeout(collectErrorTimeout);
    });

    // Reload silently when backend pushes a new package inventory.
    // seenSignal captures the value at mount time so we only react to signals
    // that arrive AFTER this component is rendered (not pre-existing ones).
    let seenSignal = ctx.packageInventorySignal;
    $effect(() => {
        const sig = ctx.packageInventorySignal;
        if (sig > seenSignal) {
            seenSignal = sig;
            awaitingInventory = false;
            try {
                sessionStorage.removeItem(COLLECT_SESSION_KEY);
            } catch {
                /* unavailable */
            }
            loadData(true);
        }
    });

    function saveToCache() {
        ctx.setPackagesCache({
            allPackages,
            stats,
            searchTerm,
            allManagerKeys,
            selectedManagers: [...selectedManagers],
            visibleColumns: [...visibleColumns],
        });
    }

    async function loadData(silent = false) {
        if (!silent) loading = true;
        try {
            const [packagesData, statsData] = await Promise.all([
                api.getHostPackages(hostId),
                api.getPackageStats(hostId),
            ]);

            allPackages = packagesData.packages || [];
            stats = statsData;

            if (
                allManagerKeys.length === 0 &&
                statsData.by_package_manager?.length > 0
            ) {
                allManagerKeys = statsData.by_package_manager.map(
                    (pm: { package_manager: string }) => pm.package_manager,
                );
            }
        } catch (err: unknown) {
            if (!silent)
                error =
                    err instanceof Error
                        ? err.message
                        : "Failed to load packages";
        } finally {
            if (!silent) loading = false;
            saveToCache();
        }
    }

    let searchDebounce: ReturnType<typeof setTimeout> | null = null;

    function handleSearchInput() {
        offset = 0;
        clearTimeout(searchDebounce);
        searchDebounce = setTimeout(() => saveToCache(), 300);
    }

    function toggleManager(manager: string) {
        const next = new Set(selectedManagers);
        if (next.has(manager)) {
            next.delete(manager);
        } else {
            next.add(manager);
        }
        selectedManagers = next;
        offset = 0;
    }

    function clearFilter() {
        selectedManagers = new Set();
        offset = 0;
    }

    const hasActiveFilters = $derived(
        searchTerm !== "" || quickFilter !== "all" || selectedManagers.size > 0,
    );

    function clearAllFilters() {
        searchTerm = "";
        quickFilter = "all";
        selectedManagers = new Set();
        offset = 0;
    }

    function toggleColumn(key: ColumnKey) {
        const next = new Set(visibleColumns);
        if (next.has(key)) {
            next.delete(key);
        } else {
            next.add(key);
        }
        visibleColumns = next;
        saveToCache();
    }

    function setQuickFilter(
        filter: "all" | "needs_update" | "outdated" | "security",
    ) {
        quickFilter = filter;
        offset = 0;
    }

    function handlePageChange(newPage: number) {
        offset = (newPage - 1) * limit;
    }

    function handlePageSizeChange(size: number) {
        limit = size;
        offset = 0;
    }

    function handleSort(column: string) {
        if (sortColumn === column) {
            sortOrder = sortOrder === "asc" ? "desc" : "asc";
        } else {
            sortColumn = column;
            // For latest_version, default to desc so packages with updates appear first
            sortOrder = column === "latest_version" ? "desc" : "asc";
        }
        offset = 0;
    }

    let collecting = $state(false);
    let collectError = $state("");
    let collectErrorTimeout: ReturnType<typeof setTimeout> | null = null;

    const COLLECT_SESSION_KEY = $derived(`wf_awaiting_collect_${hostId}`);
    const COLLECT_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes

    function getStoredAwaitingInventory(): boolean {
        try {
            const raw = sessionStorage.getItem(COLLECT_SESSION_KEY);
            if (!raw) return false;
            const ts = Number(raw);
            if (isNaN(ts) || Date.now() - ts > COLLECT_TIMEOUT_MS) {
                sessionStorage.removeItem(COLLECT_SESSION_KEY);
                return false;
            }
            return true;
        } catch {
            return false;
        }
    }

    let awaitingInventory = $state(getStoredAwaitingInventory());

    async function handleForceCollect() {
        if (collecting) return;
        collecting = true;
        collectError = "";
        try {
            await api.triggerPackageCollect(hostId);
            awaitingInventory = true;
            try {
                sessionStorage.setItem(COLLECT_SESSION_KEY, String(Date.now()));
            } catch {
                /* unavailable */
            }
        } catch (err: unknown) {
            collectError =
                err instanceof Error
                    ? err.message
                    : "Failed to trigger collection";
            if (collectErrorTimeout) clearTimeout(collectErrorTimeout);
            collectErrorTimeout = setTimeout(() => {
                collectError = "";
            }, 4000);
        } finally {
            collecting = false;
        }
    }
</script>

<svelte:head>
    <title
        >Packages{ctx.host ? ` - ${ctx.host.display_name}` : ""} - Watchflare</title
    >
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
    <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-4">
        {#each Array(3) as _}
            <div class="rounded-xl border bg-card px-4 py-3.5 animate-pulse">
                <div class="h-3 w-20 rounded bg-muted mb-3"></div>
                <div class="h-7 w-12 rounded bg-muted"></div>
            </div>
        {/each}
    </div>
{:else if stats}
    <div class="flex items-center gap-x-4 gap-y-1 flex-wrap text-sm mb-4">
        <span class="text-muted-foreground"
            ><span class="font-medium text-foreground"
                >{stats.total_packages || 0}</span
            > packages</span
        >
        {#if (stats.outdated_count || 0) > 0}
            <span class="text-amber-500"
                ><span class="font-medium">{stats.outdated_count}</span> outdated</span
            >
        {/if}
        {#if (stats.security_updates_count || 0) > 0}
            <span class="text-destructive"
                ><span class="font-medium">{stats.security_updates_count}</span> security
                updates</span
            >
        {/if}
    </div>
{/if}

<!-- Search & Filters -->
<div class="mb-4 flex flex-col gap-2">
    <!-- Row 1: search + collect -->
    <div class="flex items-center gap-2">
        <input
            type="text"
            bind:value={searchTerm}
            oninput={handleSearchInput}
            onkeydown={(e) => {
                if (e.key === "Enter") {
                    clearTimeout(searchDebounce);
                    offset = 0;
                    saveToCache();
                }
            }}
            placeholder="Search packages..."
            class="flex-1 min-w-0 h-9 rounded-lg border bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
        />
        <button
            onclick={handleForceCollect}
            disabled={collecting ||
                awaitingInventory ||
                ctx.host?.status !== "online"}
            title={ctx.host?.status !== "online"
                ? "Host must be online to collect packages"
                : "Force package collection now"}
            class="shrink-0 inline-flex items-center gap-1.5 h-9 rounded-lg border px-3 text-sm font-medium transition-colors whitespace-nowrap
                bg-card text-muted-foreground hover:bg-muted hover:text-foreground
                disabled:opacity-40 disabled:cursor-not-allowed"
        >
            <RefreshCw
                class="h-3.5 w-3.5 {collecting || awaitingInventory
                    ? 'animate-spin'
                    : ''}"
            />
            <span class="hidden sm:inline">Collect Now</span>
        </button>
    </div>

    <!-- Row 2: status indicator (left) + filters (right) -->
    <div class="flex items-center gap-2">
        <div class="flex-1 text-xs text-muted-foreground">
            {#if collectError}
                <span class="text-destructive">{collectError}</span>
            {/if}
        </div>

        {#if hasActiveFilters}
            <button
                onclick={clearAllFilters}
                class="inline-flex items-center gap-1.5 h-9 rounded-lg border px-3 text-sm font-medium transition-colors whitespace-nowrap bg-card text-muted-foreground hover:bg-muted hover:text-foreground"
                title="Clear all filters"
            >
                <X class="h-3.5 w-3.5 shrink-0" />
                <span class="hidden sm:inline">Clear filters</span>
            </button>
        {/if}

        <!-- Status filter -->
        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        class="inline-flex items-center gap-1.5 h-9 rounded-lg border px-3 text-sm font-medium transition-colors whitespace-nowrap
                        {quickFilter === 'security'
                            ? 'border-destructive/40 bg-destructive/5 text-destructive'
                            : quickFilter !== 'all'
                              ? 'border-amber-500/40 bg-amber-500/5 text-amber-500'
                              : 'bg-card text-muted-foreground hover:bg-muted hover:text-foreground'}"
                    >
                        {#if quickFilter === "security"}
                            <ShieldAlert class="h-3.5 w-3.5 shrink-0" />
                        {:else if quickFilter !== "all"}
                            <ArrowUp class="h-3.5 w-3.5 shrink-0" />
                        {:else}
                            <Tag class="h-3.5 w-3.5 shrink-0" />
                        {/if}
                        <span class="hidden sm:inline">
                            {quickFilter === "all"
                                ? "All statuses"
                                : quickFilter === "needs_update"
                                  ? "Needs update"
                                  : quickFilter === "outdated"
                                    ? "Outdated only"
                                    : "Security updates"}
                        </span>
                        <ChevronDown
                            class="hidden sm:inline-block h-3 w-3 opacity-40"
                        />
                    </button>
                {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end" class="group">
                {#each [{ value: "all", label: "All statuses" }, { value: "needs_update", label: "Needs update" }, { value: "outdated", label: "Outdated only" }, { value: "security", label: "Security updates" }] as option}
                    <DropdownMenu.Item
                        closeOnSelect={true}
                        onclick={() =>
                            setQuickFilter(option.value as typeof quickFilter)}
                        class={quickFilter === option.value
                            ? "bg-muted font-medium group-has-data-highlighted:bg-transparent data-highlighted:bg-muted"
                            : ""}
                    >
                        {option.label}
                    </DropdownMenu.Item>
                {/each}
            </DropdownMenu.Content>
        </DropdownMenu.Root>

        <!-- Package manager filter -->
        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        class="inline-flex items-center gap-1.5 h-9 rounded-lg border px-3 text-sm font-medium transition-colors whitespace-nowrap
                        {isFiltered
                            ? 'border-primary/40 bg-primary/5 text-primary hover:bg-primary/10'
                            : 'bg-card text-muted-foreground hover:bg-muted hover:text-foreground'}"
                    >
                        <Filter class="h-3.5 w-3.5" />
                        <span class="hidden sm:inline">{filterLabel}</span>
                        {#if isFiltered}
                            <span
                                class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-primary/15 px-1 text-xs font-medium text-primary"
                            >
                                {selectedManagers.size}
                            </span>
                        {/if}
                        <ChevronDown
                            class="hidden sm:inline-block h-3 w-3 opacity-40"
                        />
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

        <!-- Column visibility (desktop only) -->
        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        class="hidden md:inline-flex items-center gap-1.5 h-9 rounded-lg border px-3 text-sm font-medium transition-colors whitespace-nowrap
                        {extraColumnsCount > 0
                            ? 'border-primary/40 bg-primary/5 text-primary hover:bg-primary/10'
                            : 'bg-card text-muted-foreground hover:bg-muted hover:text-foreground'}"
                    >
                        <Columns3 class="h-3.5 w-3.5" />
                        <span class="hidden sm:inline">Columns</span>
                        {#if extraColumnsCount > 0}
                            <span
                                class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-primary/15 px-1 text-xs font-medium text-primary"
                            >
                                +{extraColumnsCount}
                            </span>
                        {/if}
                        <ChevronDown
                            class="hidden sm:inline-block h-3 w-3 opacity-40"
                        />
                    </button>
                {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end">
                {#each ALL_COLUMNS as column}
                    <DropdownMenu.Item
                        closeOnSelect={false}
                        onclick={() => toggleColumn(column.key)}
                    >
                        <div
                            class="flex h-4 w-4 shrink-0 items-center justify-center rounded border
                        {visibleColumns.has(column.key)
                                ? 'border-primary bg-primary'
                                : 'border-muted-foreground/40'}"
                        >
                            {#if visibleColumns.has(column.key)}
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
                        <span class="flex-1">{column.label}</span>
                    </DropdownMenu.Item>
                {/each}
            </DropdownMenu.Content>
        </DropdownMenu.Root>
    </div>
    <!-- end row 2 -->
</div>

{#snippet sortIcon(column: string)}
    {#if sortColumn === column}
        <svg class="h-3 w-3 shrink-0" viewBox="0 0 12 12" fill="currentColor">
            {#if sortOrder === "asc"}
                <path d="M6 2l4 5H2z" />
            {:else}
                <path d="M6 10l4-5H2z" />
            {/if}
        </svg>
    {:else}
        <svg
            class="h-3 w-3 shrink-0 opacity-0 group-hover:opacity-50 transition-opacity"
            viewBox="0 0 12 12"
            fill="currentColor"
        >
            <path d="M6 10l4-5H2z" />
        </svg>
    {/if}
{/snippet}

<!-- Packages Table/Cards -->
<div class="rounded-xl border bg-card overflow-hidden mb-6">
    {#if loading}
        <div class="flex items-center justify-center py-20">
            <p class="text-muted-foreground">Loading packages...</p>
        </div>
    {:else}
        <!-- Mobile: cards -->
        <div class="md:hidden p-3 flex flex-col gap-2">
            {#each paginatedPackages() as pkg}
                <div class="rounded-lg border bg-card">
                    <!-- Header: name + status badge -->
                    <div
                        class="rounded-t-lg bg-muted/30 px-4 py-2.5 border-b border-border flex items-center justify-between gap-2"
                    >
                        <span class="flex items-center gap-2 min-w-0">
                            <PackageIcon
                                class="h-3.5 w-3.5 shrink-0 text-muted-foreground"
                            />
                            <span
                                class="text-sm font-medium text-foreground break-all"
                                >{pkg.name}</span
                            >
                        </span>
                        {#if pkg.has_security_update}
                            <span
                                class="shrink-0 inline-flex rounded-full border px-2 py-0.5 text-xs font-medium bg-destructive/10 text-destructive border-destructive/20"
                                >Security Update</span
                            >
                        {:else if pkg.available_version}
                            <span
                                class="shrink-0 inline-flex rounded-full border px-2 py-0.5 text-xs font-medium bg-amber-500/10 text-amber-500 border-amber-500/20"
                                >Outdated</span
                            >
                        {:else}
                            <span
                                class="shrink-0 inline-flex rounded-full border px-2 py-0.5 text-xs font-medium bg-success/10 text-success border-success/20"
                                >Up to date</span
                            >
                        {/if}
                    </div>
                    <!-- Body: version + latest + manager -->
                    <div class="px-4 py-2.5 flex items-center gap-2 flex-wrap">
                        <span class="text-xs font-mono text-muted-foreground"
                            >{pkg.version || "—"}</span
                        >
                        {#if pkg.available_version}
                            <span
                                class="inline-flex items-center gap-1 text-xs font-mono font-medium {pkg.has_security_update
                                    ? 'text-destructive'
                                    : 'text-amber-500'}"
                            >
                                {#if pkg.has_security_update}
                                    <ShieldAlert class="h-3 w-3 shrink-0" />
                                {:else}
                                    <ArrowUp class="h-3 w-3 shrink-0" />
                                {/if}
                                {pkg.available_version}
                            </span>
                        {/if}
                        <span
                            class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getManagerColor(
                                pkg.package_manager,
                            )}">{getManagerLabel(pkg.package_manager)}</span
                        >
                    </div>
                </div>
            {:else}
                <div class="py-16 text-center">
                    <svg
                        class="mx-auto h-10 w-10 text-muted-foreground/40 mb-3"
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
                    <p class="text-sm text-muted-foreground">
                        No packages found
                    </p>
                </div>
            {/each}
        </div>

        <!-- Desktop: table -->
        <div class="hidden md:block overflow-auto max-h-[65vh]">
            <table class="w-full min-w-120">
                <thead>
                    <tr
                        class="bg-table-header sticky top-0 z-10 [box-shadow:0_1px_0_var(--border)]"
                    >
                        {#if col("name")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground"
                            >
                                <button
                                    type="button"
                                    onclick={() => handleSort("name")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground {sortColumn ===
                                    'name'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Name {@render sortIcon("name")}</button
                                >
                            </th>
                        {/if}
                        {#if col("version")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground"
                            >
                                <button
                                    type="button"
                                    onclick={() => handleSort("version")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground {sortColumn ===
                                    'version'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Version {@render sortIcon(
                                        "version",
                                    )}</button
                                >
                            </th>
                        {/if}
                        {#if col("status")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground w-px whitespace-nowrap"
                            >
                                <button type="button"
                                    onclick={() => handleSort("status")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground {sortColumn ===
                                    'status'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Status {@render sortIcon("status")}</button
                                >
                            </th>
                        {/if}
                        {#if col("manager")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground w-px whitespace-nowrap"
                            >
                                <button type="button"
                                    onclick={() => handleSort("manager")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground {sortColumn ===
                                    'manager'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Manager {@render sortIcon(
                                        "manager",
                                    )}</button
                                >
                            </th>
                        {/if}
                        {#if col("latest_version")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground whitespace-nowrap"
                            >
                                <button type="button"
                                    onclick={() => handleSort("latest_version")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground {sortColumn ===
                                    'latest_version'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Latest Version {@render sortIcon(
                                        "latest_version",
                                    )}</button
                                >
                            </th>
                        {/if}
                        {#if col("arch")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground w-28"
                            >
                                <button type="button"
                                    onclick={() => handleSort("arch")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground {sortColumn ===
                                    'arch'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Architecture {@render sortIcon(
                                        "arch",
                                    )}</button
                                >
                            </th>
                        {/if}
                        {#if col("description")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-left text-sm font-semibold text-muted-foreground w-64"
                                >Description</th
                            >
                        {/if}
                        {#if col("first_seen")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-right text-sm font-semibold text-muted-foreground w-36"
                            >
                                <button type="button"
                                    onclick={() => handleSort("first_seen")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground ml-auto {sortColumn ===
                                    'first_seen'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >First Seen {@render sortIcon(
                                        "first_seen",
                                    )}</button
                                >
                            </th>
                        {/if}
                        {#if col("last_seen")}
                            <th
                                scope="col"
                                class="px-4 py-2 text-right text-sm font-semibold text-muted-foreground w-36"
                            >
                                <button type="button"
                                    onclick={() => handleSort("last_seen")}
                                    class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-table-header-active hover:text-foreground ml-auto {sortColumn ===
                                    'last_seen'
                                        ? 'bg-table-header-active text-foreground'
                                        : ''}"
                                    >Last Seen {@render sortIcon(
                                        "last_seen",
                                    )}</button
                                >
                            </th>
                        {/if}
                    </tr>
                </thead>
                <tbody class="divide-y divide-border">
                    {#each paginatedPackages() as pkg}
                        <tr class="hover:bg-muted/20 transition-colors">
                            {#if col("name")}
                                <td class="px-4 py-3">
                                    <span class="flex items-center gap-2">
                                        <PackageIcon
                                            class="h-3.5 w-3.5 shrink-0 text-muted-foreground"
                                        />
                                        <span
                                            class="text-sm font-medium text-foreground"
                                            >{pkg.name}</span
                                        >
                                    </span>
                                </td>
                            {/if}
                            {#if col("version")}
                                <td
                                    class="px-4 py-3 text-sm font-mono text-muted-foreground whitespace-nowrap"
                                    >{pkg.version || "-"}</td
                                >
                            {/if}
                            {#if col("status")}
                                <td class="px-4 py-3 w-px whitespace-nowrap">
                                    {#if pkg.has_security_update}
                                        <span
                                            class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium bg-destructive/10 text-destructive border-destructive/20"
                                            >Security Update</span
                                        >
                                    {:else if pkg.available_version}
                                        <span
                                            class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium bg-amber-500/10 text-amber-500 border-amber-500/20"
                                            >Outdated</span
                                        >
                                    {:else}
                                        <span
                                            class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium bg-success/10 text-success border-success/20"
                                            >Up to date</span
                                        >
                                    {/if}
                                </td>
                            {/if}
                            {#if col("manager")}
                                <td class="px-4 py-3 w-px whitespace-nowrap">
                                    <span
                                        class="inline-flex rounded-full border px-2 py-0.5 text-xs font-medium {getManagerColor(
                                            pkg.package_manager,
                                        )}"
                                    >
                                        {getManagerLabel(pkg.package_manager)}
                                    </span>
                                </td>
                            {/if}
                            {#if col("latest_version")}
                                <td class="px-4 py-3 text-sm whitespace-nowrap">
                                    {#if pkg.available_version}
                                        <span
                                            class="inline-flex items-center gap-1"
                                        >
                                            {#if pkg.has_security_update}
                                                <ShieldAlert
                                                    class="h-3.5 w-3.5 text-destructive shrink-0"
                                                />
                                                <span
                                                    class="font-mono font-medium text-destructive"
                                                    >{pkg.available_version}</span
                                                >
                                            {:else}
                                                <ArrowUp
                                                    class="h-3.5 w-3.5 text-amber-500 shrink-0"
                                                />
                                                <span
                                                    class="font-mono font-medium text-amber-500"
                                                    >{pkg.available_version}</span
                                                >
                                            {/if}
                                        </span>
                                    {:else}
                                        <span class="text-muted-foreground/40"
                                            >—</span
                                        >
                                    {/if}
                                </td>
                            {/if}
                            {#if col("arch")}
                                <td
                                    class="px-4 py-3 text-sm text-muted-foreground whitespace-nowrap"
                                    >{pkg.architecture || "-"}</td
                                >
                            {/if}
                            {#if col("description")}
                                <td class="px-4 py-3">
                                    <span
                                        class="block text-sm text-muted-foreground truncate max-w-64"
                                        title={pkg.description || ""}
                                        >{pkg.description || "-"}</span
                                    >
                                </td>
                            {/if}
                            {#if col("first_seen")}
                                <td
                                    class="px-4 py-3 text-right text-sm text-muted-foreground whitespace-nowrap"
                                    >{formatDateTime(
                                        pkg.first_seen,
                                        timeFormat,
                                    )}</td
                                >
                            {/if}
                            {#if col("last_seen")}
                                <td
                                    class="px-4 py-3 text-right text-sm text-muted-foreground whitespace-nowrap"
                                    >{formatDateTime(
                                        pkg.last_seen,
                                        timeFormat,
                                    )}</td
                                >
                            {/if}
                        </tr>
                    {:else}
                        <tr>
                            <td
                                colspan={visibleColumns.size}
                                class="py-16 text-center"
                            >
                                <svg
                                    class="mx-auto h-10 w-10 text-muted-foreground/40 mb-3"
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
                                <p class="text-sm text-muted-foreground">
                                    No packages found
                                </p>
                            </td>
                        </tr>
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
            onPageSizeChange={handlePageSizeChange}
            pageSizeOptions={PAGE_SIZE_OPTIONS}
        />
    {/if}
</div>
