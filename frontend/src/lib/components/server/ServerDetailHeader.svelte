<script lang="ts">
    import {
        getStatusClass,
        formatRelativeTime,
        formatUptime,
    } from "$lib/utils";
    import {
        Pause,
        Play,
        Pencil,
        Globe,
        RefreshCw,
        Trash2,
        EllipsisVertical,
        Server as ServerIcon,
        Monitor,
        Cpu,
        Network,
        Clock,
    } from "lucide-svelte";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
    import type { Server, GetPackageStatsResponse, Metric } from "$lib/types";

    const {
        server,
        packageStats,
        metric = null,
        onDelete,
        onRegenerateToken,
        onChangeIP,
        onRename,
        onPause,
        onResume,
    }: {
        server: Server;
        packageStats: GetPackageStatsResponse | null;
        metric?: Metric | null;
        onDelete: () => void;
        onRegenerateToken: () => void;
        onChangeIP: () => void;
        onRename: () => void;
        onPause: () => void;
        onResume: () => void;
    } = $props();

    let open = $state(false);

    const details = $derived(
        [
            server.hostname
                ? { icon: ServerIcon, text: server.hostname }
                : null,
            server.platform
                ? {
                      icon: Monitor,
                      text: server.platform_version
                          ? `${server.platform} ${server.platform_version}`
                          : server.platform,
                  }
                : null,
            server.architecture
                ? { icon: Cpu, text: server.architecture }
                : null,
            server.ip_address_v4 || server.configured_ip
                ? {
                      icon: Network,
                      text: server.ip_address_v4 || server.configured_ip,
                  }
                : null,
            metric && metric.uptime_seconds > 0
                ? { icon: Clock, text: formatUptime(metric.uptime_seconds) }
                : null,
            server.last_seen &&
            (server.status === "offline" || server.status === "paused")
                ? {
                      icon: Clock,
                      text: `Last seen ${formatRelativeTime(server.last_seen)}`,
                  }
                : null,
        ].filter(Boolean) as { icon: typeof ServerIcon; text: string }[],
    );
</script>

<div class="mb-6 rounded-lg border bg-card p-4 md:p-6">
    <!-- Top: Name, status, actions menu -->
    <div class="flex items-start justify-between">
        <div class="flex items-center gap-3 flex-wrap">
            <h1
                class="text-xl sm:text-2xl font-semibold text-foreground break-all"
            >
                {server.name}
            </h1>
            <span
                class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(
                    server.status,
                )}"
            >
                <span
                    class="h-1.5 w-1.5 rounded-full {server.status === 'online'
                        ? 'bg-success'
                        : server.status === 'offline'
                          ? 'bg-danger'
                          : 'bg-muted-foreground'}"
                ></span>
                {server.status}
            </span>
        </div>
        <DropdownMenu.Root bind:open>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        class="rounded-lg p-1.5 ml-3 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                        title="Server actions"
                    >
                        <EllipsisVertical class="h-5 w-5" />
                    </button>
                {/snippet}
            </DropdownMenu.Trigger>

            <DropdownMenu.Content side="bottom" align="end">
                <DropdownMenu.Item
                    onclick={() => {
                        open = false;
                        onRename();
                    }}
                >
                    <Pencil class="h-4 w-4" />
                    Rename
                </DropdownMenu.Item>
                <DropdownMenu.Item
                    onclick={() => {
                        open = false;
                        onChangeIP();
                    }}
                >
                    <Globe class="h-4 w-4" />
                    Change IP
                </DropdownMenu.Item>
                {#if server.status === "pending"}
                    <DropdownMenu.Item
                        onclick={() => {
                            open = false;
                            onRegenerateToken();
                        }}
                    >
                        <RefreshCw class="h-4 w-4" />
                        Regenerate Token
                    </DropdownMenu.Item>
                {/if}
                {#if server.status !== "pending"}
                    {#if server.status === "paused"}
                        <DropdownMenu.Item
                            onclick={() => {
                                open = false;
                                onResume();
                            }}
                        >
                            <Play class="h-4 w-4" />
                            Resume
                        </DropdownMenu.Item>
                    {:else}
                        <DropdownMenu.Item
                            onclick={() => {
                                open = false;
                                onPause();
                            }}
                        >
                            <Pause class="h-4 w-4" />
                            Pause
                        </DropdownMenu.Item>
                    {/if}
                {/if}
                <DropdownMenu.Separator />
                <DropdownMenu.Item
                    onclick={() => {
                        open = false;
                        onDelete();
                    }}
                    class="text-destructive data-highlighted:text-destructive"
                >
                    <Trash2 class="h-4 w-4" />
                    Delete
                </DropdownMenu.Item>
            </DropdownMenu.Content>
        </DropdownMenu.Root>
    </div>

    <!-- Server details -->
    <div
        class="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted-foreground"
    >
        {#each details as detail, i}
            {#if i > 0}
                <span>·</span>
            {/if}
            <span class="inline-flex items-center gap-1">
                <detail.icon class="h-3.5 w-3.5" />{detail.text}
            </span>
        {/each}
    </div>

    <!-- Packages -->
    {#if packageStats && packageStats.total_packages > 0}
        <div class="mt-4 pt-4 border-t">
            <div
                class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3"
            >
                <div class="flex flex-col gap-2 min-w-0">
                    <!-- Counts + last scan -->
                    <div
                        class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm"
                    >
                        <span class="font-medium text-foreground">
                            {packageStats.total_packages.toLocaleString()} packages
                        </span>
                        {#if packageStats.recent_changes}
                            <span class="text-muted-foreground">·</span>
                            <span class="text-muted-foreground">
                                {packageStats.recent_changes} changes (30d)
                            </span>
                        {/if}
                        {#if packageStats.last_collection?.timestamp}
                            <span class="text-muted-foreground">·</span>
                            <span class="text-xs text-muted-foreground">
                                Last scan {formatRelativeTime(
                                    packageStats.last_collection.timestamp,
                                )}
                            </span>
                        {/if}
                    </div>
                </div>
                <a
                    href="/servers/{server.id}/packages"
                    class="sm:shrink-0 self-start rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
                >
                    View Packages →
                </a>
            </div>
        </div>
    {/if}
</div>
