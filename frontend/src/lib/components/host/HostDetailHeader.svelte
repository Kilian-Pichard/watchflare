<script lang="ts">
    import {
        getStatusClass,
        formatUptime,
        isAgentOutdated,
        formatBytes,
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
        ClockArrowUp,
        Tag,
        ArrowUpCircle,
        Bell,
        MemoryStick,
        Download,
    } from "lucide-svelte";
    import OsIcon from "$lib/components/icons/OsIcon.svelte";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
    import type { Host, Metric } from "$lib/types";

    const {
        host,
        metric = null,
        latestAgentVersion = null,
        onDelete,
        onRegenerateToken,
        onChangeIP,
        onRename,
        onPause,
        onResume,
        onAlertRules,
        onUpdateAgent,
        hasActiveAlertRules = false,
    }: {
        host: Host;
        metric?: Metric | null;
        latestAgentVersion?: string | null;
        hasActiveAlertRules?: boolean;
        onDelete: () => void;
        onRegenerateToken: () => void;
        onChangeIP: () => void;
        onRename: () => void;
        onPause: () => void;
        onResume: () => void;
        onAlertRules: () => void;
        onUpdateAgent?: () => void;
    } = $props();

    const agentOutdated = $derived(isAgentOutdated(host.agent_version, latestAgentVersion));

    let open = $state(false);

    const details = $derived(
        [
            host.hostname
                ? { icon: ServerIcon, text: host.hostname, type: 'hostname' }
                : null,
            (host.ip_address_v4 || host.configured_ip)
                ? { icon: Network, text: (host.ip_address_v4 || host.configured_ip)!, type: 'ip' }
                : null,
            host.platform
                ? {
                      icon: Monitor,
                      text: host.platform_version
                          ? `${host.platform} ${host.platform_version}`
                          : host.platform,
                      type: 'platform',
                  }
                : null,
            host.architecture
                ? { icon: Cpu, text: host.architecture, type: 'architecture' }
                : null,
            metric && metric.memory_total_bytes > 0
                ? { icon: MemoryStick, text: formatBytes(metric.memory_total_bytes), type: 'memory' }
                : null,
            metric && metric.uptime_seconds > 0
                ? { icon: ClockArrowUp, text: formatUptime(metric.uptime_seconds), type: 'uptime' }
                : null,
            host.agent_version
                ? { icon: Tag, text: `Agent v${host.agent_version}`, type: 'agent_version' }
                : null,
        ].filter(Boolean) as { icon: typeof ServerIcon; text: string; type: string }[],
    );
</script>

<div class="mb-4 rounded-xl border bg-card p-3 md:p-4">
    <!-- Top: Name, status, actions menu -->
    <div class="flex items-start justify-between">
        <div class="flex items-center gap-3 flex-wrap">
            <h1
                class="text-xl sm:text-2xl font-semibold text-foreground break-all"
            >
                {host.display_name}
            </h1>
            <span
                class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(
                    host.status,
                )}"
            >
                <span
                    class="h-1.5 w-1.5 rounded-full {host.status === 'online'
                        ? 'bg-success'
                        : host.status === 'offline'
                          ? 'bg-danger'
                          : 'bg-muted-foreground'}"
                ></span>
                {host.status}
            </span>
        </div>
        <div class="flex items-center gap-1 ml-3">
            {#if agentOutdated && host.status === 'online' && onUpdateAgent}
                <button
                    onclick={onUpdateAgent}
                    title="Update agent to v{latestAgentVersion}"
                    class="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-amber-700 bg-amber-100 hover:bg-amber-200 dark:text-amber-400 dark:bg-amber-900/30 dark:hover:bg-amber-900/50 transition-colors"
                >
                    <Download class="h-3.5 w-3.5" />
                    Update
                </button>
            {/if}
            <button
                onclick={onAlertRules}
                class="rounded-lg p-1.5 transition-colors hover:bg-muted {hasActiveAlertRules ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}"
                title={hasActiveAlertRules ? 'Alert rules (monitoring active)' : 'Alert rules'}
            >
                <Bell class="h-5 w-5 {hasActiveAlertRules ? 'fill-current' : ''}" />
            </button>
            <DropdownMenu.Root bind:open>
                <DropdownMenu.Trigger>
                    {#snippet child({ props })}
                        <button
                            {...props}
                            class="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                            title="Host actions"
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
                    <DropdownMenu.Item
                        onclick={() => {
                            open = false;
                            onRegenerateToken();
                        }}
                    >
                        <RefreshCw class="h-4 w-4" />
                        Regenerate Token
                    </DropdownMenu.Item>
                    {#if host.status !== "pending"}
                        {#if host.status === "paused"}
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
    </div>

    <!-- Host details -->
    <div
        class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted-foreground"
    >
        {#each details as detail, i}
            {#if i > 0}
                <span aria-hidden="true">·</span>
            {/if}
            <span class="inline-flex items-center gap-1">
                {#if detail.type === 'platform'}
                    <OsIcon os={host.platform} class="h-3.5 w-3.5 shrink-0" />
                {:else}
                    <detail.icon class="h-3.5 w-3.5 shrink-0" />
                {/if}
                {detail.text}{#if agentOutdated && detail.type === 'agent_version'}&nbsp;<span class="inline-flex items-center gap-0.5 rounded-full border border-warning/20 bg-warning/10 px-1.5 py-0.5 text-xs font-medium text-warning"><ArrowUpCircle class="h-3 w-3" />v{latestAgentVersion}</span>{/if}
            </span>
        {/each}
    </div>

</div>
