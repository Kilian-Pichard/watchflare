<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { fly } from "svelte/transition";
    import { userStore } from "$lib/stores/user";
    import { Check, Send, TriangleAlert, X } from "lucide-svelte";
    import type { SmtpTLSMode, SmtpAuthType, AlertRule, AlertMetricType } from "$lib/types";
    import { ALERT_METRIC_TYPES, ALERT_METRIC_LABELS } from "$lib/types";
    import {
        getSmtpSettings,
        updateSmtpSettings,
        testSmtpConnection,
        getAlertRules,
        updateAlertRules,
    } from "$lib/api";
    import * as Select from "$lib/components/ui/select";
    import Toggle from "$lib/components/ui/Toggle.svelte";

    const user = $derived($userStore.user);

    let loaded = $state(false);
    let loadError = $state(false);
    let smtpEnabled = $state(false);
    let smtpHost = $state("");
    let smtpPort = $state(587);
    let smtpTLSMode = $state<SmtpTLSMode>("starttls");
    let smtpUsername = $state("");
    let smtpPassword = $state(""); // only sent when non-empty (change only)
    let smtpPasswordSet = $state(false); // true if a password is already stored
    let smtpAuthType = $state<SmtpAuthType>("plain");
    let smtpFromAddress = $state("");
    let smtpFromName = $state("");
    let smtpHeloName = $state("");

    let smtpSaving = $state(false);
    let smtpSaveSuccess = $state(false);
    let saveSuccessTimer: ReturnType<typeof setTimeout> | undefined;

    // Alert rules
    type EditableRule = {
        metric_type: AlertMetricType;
        enabled: boolean;
        threshold: number;
        duration_minutes: number;
    };
    let alertRules = $state<EditableRule[]>([]);
    let alertLoaded = $state(false);
    let alertSaving = $state(false);
    let alertSaveSuccess = $state(false);
    let alertSaveTimer: ReturnType<typeof setTimeout> | undefined;
    let alertSaveError = $state("");

    // Per-field validation errors (shown inline below each field)
    let fieldErrors = $state<Record<string, string>>({});

    // Toast notification for API-level save errors
    let toastMessage = $state("");
    let toastTimer: ReturnType<typeof setTimeout> | undefined;

    let testRecipient = $state("");
    let smtpTesting = $state(false);
    let smtpTestMessage = $state("");
    let smtpTestError = $state("");

    // Source: https://emailregex.com/
    const EMAIL_RE = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;

    const defaultPorts: Record<SmtpTLSMode, number> = {
        none: 25,
        starttls: 587,
        tls: 465,
    };
    const knownDefaultPorts = new Set([25, 465, 587]);

    const tlsModeOptions: { value: SmtpTLSMode; label: string }[] = [
        { value: "none", label: "None — port 25" },
        { value: "starttls", label: "STARTTLS — port 587" },
        { value: "tls", label: "SSL/TLS — port 465" },
    ];

    const selectedTLSLabel = $derived(
        tlsModeOptions.find((o) => o.value === smtpTLSMode)?.label ?? smtpTLSMode,
    );

    const authTypeOptions: { value: SmtpAuthType; label: string }[] = [
        { value: "plain", label: "PLAIN (default)" },
        { value: "login", label: "LOGIN" },
    ];

    const selectedAuthLabel = $derived(
        authTypeOptions.find((o) => o.value === smtpAuthType)?.label ?? "PLAIN (default)",
    );

    // Warn when TLS mode and port are mismatched
    const tlsPortMismatch = $derived(
        (smtpTLSMode === "starttls" && smtpPort === 465) ||
        (smtpTLSMode === "tls" && smtpPort === 587)
    );

    function onTLSModeChange(newMode: string | undefined) {
        if (!newMode) return;
        const mode = newMode as SmtpTLSMode;
        smtpTLSMode = mode;
        if (knownDefaultPorts.has(smtpPort) || isNaN(smtpPort)) {
            smtpPort = defaultPorts[mode];
        }
    }

    function showToast(msg: string) {
        toastMessage = msg;
        clearTimeout(toastTimer);
        toastTimer = setTimeout(() => { toastMessage = ""; }, 6000);
    }

    function dismissToast() {
        clearTimeout(toastTimer);
        toastMessage = "";
    }

    function formatSmtpError(err: unknown): string {
        const msg = err instanceof Error ? err.message : "An error occurred.";
        if (msg.includes("i/o timeout") || msg.includes("dial tcp") || msg.includes("connection timed out"))
            return "Connection timed out. Check the host and port.";
        if (msg.includes("connection refused"))
            return "Connection refused. The server rejected the connection on this port.";
        if (msg.includes("no such host") || msg.includes("name resolution"))
            return "Host not found. Check the SMTP hostname.";
        if (msg.includes("535") || msg.includes("authentication failed") || msg.includes("Invalid credentials"))
            return "Authentication failed. Check your username and password.";
        if (msg.includes("534") || msg.includes("must issue a STARTTLS"))
            return "The server requires STARTTLS. Switch TLS mode to STARTTLS.";
        if (msg.includes("tls") || msg.includes("TLS") || msg.includes("handshake"))
            return "TLS negotiation failed. Check your TLS mode and port combination.";
        if (msg.includes("not configured"))
            return "SMTP is not configured. Save your settings first.";
        if (msg.includes("SMTP is disabled"))
            return "Enable SMTP before sending a test email.";
        return msg;
    }

    onDestroy(() => {
        clearTimeout(saveSuccessTimer);
        clearTimeout(toastTimer);
        clearTimeout(alertSaveTimer);
    });

    onMount(async () => {
        try {
            const res = await getSmtpSettings();
            const s = res.smtp;
            smtpEnabled = s.enabled;
            smtpHost = s.host;
            smtpPort = s.port;
            smtpTLSMode = s.tls_mode;
            smtpUsername = s.username;
            smtpPasswordSet = s.password_set;
            smtpAuthType = s.auth_type || "plain";
            smtpFromAddress = s.from_address;
            smtpFromName = s.from_name;
            smtpHeloName = s.helo_name;
        } catch {
            loadError = true;
        }
        testRecipient = user?.email ?? "";
        loaded = true;

        try {
            const res = await getAlertRules();
            alertRules = res.rules.map((r) => ({
                metric_type: r.metric_type,
                enabled: r.enabled,
                threshold: r.threshold,
                duration_minutes: r.duration_minutes,
            }));
        } catch {
            // Start with empty editable rules seeded from canonical order
            alertRules = ALERT_METRIC_TYPES.map((mt) => ({
                metric_type: mt,
                enabled: false,
                threshold: mt === 'temperature' ? 80 : mt === 'load_avg' ? 5 : 90,
                duration_minutes: mt === 'server_down' ? 1 : 5,
            }));
        }
        alertLoaded = true;
    });

    async function handleSaveAlerts() {
        alertSaveError = "";
        alertSaveSuccess = false;
        clearTimeout(alertSaveTimer);

        const invalidThreshold = alertRules.find(r => r.enabled && r.metric_type !== 'server_down' && (r.threshold === null || isNaN(r.threshold as unknown as number)));
        if (invalidThreshold) { alertSaveError = `Enter a threshold for ${ALERT_METRIC_LABELS[invalidThreshold.metric_type]}.`; return; }
        const invalidDuration = alertRules.find(r => r.enabled && (r.duration_minutes === null || isNaN(r.duration_minutes as unknown as number)));
        if (invalidDuration) { alertSaveError = `Enter a duration for ${ALERT_METRIC_LABELS[invalidDuration.metric_type]}.`; return; }

        alertSaving = true;
        try {
            await updateAlertRules(alertRules);
            alertSaveSuccess = true;
            alertSaveTimer = setTimeout(() => { alertSaveSuccess = false; }, 3000);
        } catch (err) {
            alertSaveError = err instanceof Error ? err.message : "Failed to save alert rules.";
        } finally {
            alertSaving = false;
        }
    }

    function thresholdUnit(metricType: AlertMetricType): string {
        if (metricType === 'cpu_usage' || metricType === 'memory_usage' || metricType === 'disk_usage') return '%';
        if (metricType === 'temperature') return '°C';
        return '';
    }

    const ALERT_DEFAULTS: Record<AlertMetricType, { threshold: number; duration: number }> = {
        server_down:   { threshold: 0,  duration: 1 },
        cpu_usage:     { threshold: 90, duration: 5 },
        memory_usage:  { threshold: 90, duration: 5 },
        disk_usage:    { threshold: 90, duration: 5 },
        load_avg:      { threshold: 5,  duration: 5 },
        load_avg_5:    { threshold: 5,  duration: 5 },
        load_avg_15:   { threshold: 5,  duration: 5 },
        temperature:   { threshold: 80, duration: 5 },
    };

    async function handleSave() {
        fieldErrors = {};
        smtpSaveSuccess = false;
        clearTimeout(saveSuccessTimer);

        if (smtpEnabled) {
            if (!smtpFromName) fieldErrors.fromName = "Cannot be blank";
            if (!smtpFromAddress) fieldErrors.fromAddress = "Cannot be blank";
            else if (!EMAIL_RE.test(smtpFromAddress)) fieldErrors.fromAddress = "Invalid email format";
            if (!smtpHost) fieldErrors.host = "Cannot be blank";
            if (!smtpPort || isNaN(smtpPort) || smtpPort < 1 || smtpPort > 65535)
                fieldErrors.port = "Must be between 1 and 65535";
            if (Object.keys(fieldErrors).length > 0) return;
        }

        smtpSaving = true;
        try {
            await updateSmtpSettings({
                host: smtpHost,
                port: smtpPort,
                username: smtpUsername,
                password: smtpPassword,
                from_address: smtpFromAddress,
                from_name: smtpFromName,
                tls_mode: smtpTLSMode,
                auth_type: smtpAuthType,
                helo_name: smtpHeloName,
                enabled: smtpEnabled,
            });
            if (smtpPassword) {
                smtpPasswordSet = true;
                smtpPassword = "";
            }
            smtpSaveSuccess = true;
            saveSuccessTimer = setTimeout(() => { smtpSaveSuccess = false; }, 3000);
        } catch (err) {
            showToast(formatSmtpError(err));
        } finally {
            smtpSaving = false;
        }
    }

    async function handleTest() {
        smtpTestMessage = "";
        smtpTestError = "";
        if (!EMAIL_RE.test(testRecipient)) {
            smtpTestError = "Enter a valid recipient email address.";
            return;
        }
        smtpTesting = true;
        try {
            const res = await testSmtpConnection(testRecipient);
            smtpTestMessage = res.message;
        } catch (err) {
            smtpTestError = formatSmtpError(err);
        } finally {
            smtpTesting = false;
        }
    }
</script>

<svelte:head>
    <title>Notifications - Watchflare</title>
</svelte:head>

<div class="mb-6">
    <h1 class="text-xl sm:text-2xl font-semibold text-foreground">Notifications</h1>
    <p class="text-sm text-muted-foreground mt-1">Email notification settings</p>
</div>

{#if loadError}
    <div role="alert" class="mb-4 flex items-start gap-2 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2.5">
        <TriangleAlert class="h-4 w-4 text-destructive shrink-0 mt-0.5" />
        <p class="text-sm text-destructive">Failed to load SMTP settings. Your changes may overwrite the existing configuration.</p>
    </div>
{/if}

<div
    class="rounded-lg border bg-card p-4 sm:p-6 mb-6 transition-opacity duration-200 {loaded ? 'opacity-100' : 'opacity-0'}"
>
    <h2 class="text-lg font-semibold text-foreground mb-6">SMTP</h2>

    <!-- Enabled toggle -->
    <div class="mb-6 flex items-center gap-3">
        <Toggle bind:checked={smtpEnabled} aria-labelledby="smtp-enable-label" />
        <div>
            <p id="smtp-enable-label" class="text-sm font-medium text-foreground">Email notifications</p>
            <p class="text-xs text-muted-foreground mt-0.5">Send alert emails via SMTP</p>
        </div>
    </div>

    <!-- Sender Name + Sender Email -->
    <div class="mb-6 flex flex-col sm:flex-row gap-4">
        <div class="flex-1">
            <label for="smtp-from-name" class="block text-sm font-medium text-foreground mb-1">Sender Name<span class="text-destructive ml-0.5">*</span></label>
            <input
                id="smtp-from-name"
                type="text"
                placeholder="Watchflare"
                bind:value={smtpFromName}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 {fieldErrors.fromName ? 'border-destructive focus-visible:ring-destructive' : 'focus-visible:ring-primary'}"
            />
            {#if fieldErrors.fromName}
                <p class="mt-1 text-xs text-destructive">{fieldErrors.fromName}</p>
            {/if}
        </div>
        <div class="flex-1">
            <label for="smtp-from-address" class="block text-sm font-medium text-foreground mb-1">
                Sender Email<span class="text-destructive ml-0.5">*</span>
            </label>
            <input
                id="smtp-from-address"
                type="email"
                placeholder="noreply@example.com"
                bind:value={smtpFromAddress}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 {fieldErrors.fromAddress ? 'border-destructive focus-visible:ring-destructive' : 'focus-visible:ring-primary'}"
            />
            {#if fieldErrors.fromAddress}
                <p class="mt-1 text-xs text-destructive">{fieldErrors.fromAddress}</p>
            {/if}
        </div>
    </div>

    <!-- Host + Port -->
    <div class="mb-6 flex flex-col sm:flex-row gap-4">
        <div class="flex-1">
            <label for="smtp-host" class="block text-sm font-medium text-foreground mb-1">
                Host<span class="text-destructive ml-0.5">*</span>
            </label>
            <input
                id="smtp-host"
                type="text"
                placeholder="smtp.example.com"
                bind:value={smtpHost}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 {fieldErrors.host ? 'border-destructive focus-visible:ring-destructive' : 'focus-visible:ring-primary'}"
            />
            {#if fieldErrors.host}
                <p class="mt-1 text-xs text-destructive">{fieldErrors.host}</p>
            {/if}
        </div>
        <div class="w-28">
            <label for="smtp-port" class="block text-sm font-medium text-foreground mb-1">Port<span class="text-destructive ml-0.5">*</span></label>
            <input
                id="smtp-port"
                type="number"
                min="1"
                max="65535"
                bind:value={smtpPort}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus-visible:ring-2 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none {fieldErrors.port ? 'border-destructive focus-visible:ring-destructive' : 'focus-visible:ring-primary'}"
            />
            {#if fieldErrors.port}
                <p class="mt-1 text-xs text-destructive">{fieldErrors.port}</p>
            {/if}
        </div>
    </div>

    <!-- Username + Password -->
    <div class="mb-6 flex flex-col sm:flex-row gap-4">
        <div class="flex-1">
            <label for="smtp-username" class="block text-sm font-medium text-foreground mb-1">Username</label>
            <input
                id="smtp-username"
                type="text"
                placeholder="user@example.com"
                bind:value={smtpUsername}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            />
        </div>
        <div class="flex-1">
            <label for="smtp-password" class="block text-sm font-medium text-foreground mb-1">Password</label>
            <input
                id="smtp-password"
                type="password"
                autocomplete="new-password"
                placeholder={smtpPasswordSet ? "Update password" : "Password"}
                bind:value={smtpPassword}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            />
        </div>
    </div>

    <!-- TLS Mode + Auth Type + HELO -->
    <div class="mb-6 flex flex-col sm:flex-row sm:flex-wrap lg:flex-nowrap gap-6">
        <div class="shrink-0">
            <p id="tls-mode-label" class="text-sm font-medium text-foreground mb-1">TLS Mode</p>
            <p class="text-xs text-muted-foreground mb-3">Encryption method</p>
            <div class="w-48">
                <Select.Root type="single" value={smtpTLSMode} onValueChange={onTLSModeChange}>
                    <Select.Trigger aria-labelledby="tls-mode-label" items={tlsModeOptions.map((o) => o.label)}>
                        <span>{selectedTLSLabel}</span>
                    </Select.Trigger>
                    <Select.Content>
                        {#each tlsModeOptions as opt}
                            <Select.Item value={opt.value} label={opt.label}>
                                {opt.label}
                            </Select.Item>
                        {/each}
                    </Select.Content>
                </Select.Root>
            </div>
        </div>

        <div class="shrink-0">
            <p id="auth-type-label" class="text-sm font-medium text-foreground mb-1">Authentication</p>
            <p class="text-xs text-muted-foreground mb-3">Auth mechanism</p>
            <div class="w-48">
                <Select.Root
                    type="single"
                    value={smtpAuthType}
                    onValueChange={(v) => { if (v) smtpAuthType = v as SmtpAuthType; }}
                >
                    <Select.Trigger aria-labelledby="auth-type-label" items={authTypeOptions.map((o) => o.label)}>
                        <span>{selectedAuthLabel}</span>
                    </Select.Trigger>
                    <Select.Content>
                        {#each authTypeOptions as opt}
                            <Select.Item value={opt.value} label={opt.label}>
                                {opt.label}
                            </Select.Item>
                        {/each}
                    </Select.Content>
                </Select.Root>
            </div>
        </div>

        <div class="w-full lg:flex-1 lg:w-auto">
            <label for="smtp-helo-name" class="text-sm font-medium text-foreground mb-1 block">
                HELO/EHLO Hostname <span class="text-xs font-normal text-muted-foreground">(optional)</span>
            </label>
            <p class="text-xs text-muted-foreground mb-3">Leave empty to use the system hostname</p>
            <input
                id="smtp-helo-name"
                type="text"
                placeholder="mail.example.com"
                bind:value={smtpHeloName}
                class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            />
        </div>
    </div>

    <!-- TLS / port mismatch warning -->
    {#if tlsPortMismatch}
        <div class="mb-4 flex items-start gap-2 rounded-lg border border-warning/50 bg-warning/10 px-3 py-2.5">
            <TriangleAlert class="h-4 w-4 text-warning shrink-0 mt-0.5" />
            <p class="text-sm text-warning">
                {#if smtpTLSMode === "starttls" && smtpPort === 465}
                    Port 465 is for SSL/TLS, not STARTTLS. Consider switching TLS mode to SSL/TLS or changing the port to 587.
                {:else}
                    Port 587 is for STARTTLS, not SSL/TLS. Consider switching TLS mode to STARTTLS or changing the port to 465.
                {/if}
            </p>
        </div>
    {/if}

    <button
        type="button"
        onclick={handleSave}
        disabled={smtpSaving || smtpSaveSuccess}
        class="flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
    >
        {#if smtpSaveSuccess}
            <Check class="h-4 w-4" />
            Saved
        {:else}
            {smtpSaving ? "Saving..." : "Save SMTP settings"}
        {/if}
    </button>

    <!-- Test email -->
    <div class="mt-6 pt-6 border-t border-border">
        <p class="block text-sm font-medium text-foreground mb-1">Send test email</p>
        <p class="text-xs text-muted-foreground mb-3">
            Verify your SMTP configuration by sending a test email
        </p>
        <div class="flex gap-3 items-center flex-wrap">
            <input
                type="email"
                aria-label="Test recipient email address"
                placeholder="recipient@example.com"
                bind:value={testRecipient}
                class="w-64 rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            />
            <button
                type="button"
                onclick={handleTest}
                disabled={smtpTesting || !smtpEnabled || !EMAIL_RE.test(testRecipient)}
                class="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
            >
                <Send class="h-4 w-4" />
                {smtpTesting ? "Sending..." : "Send test email"}
            </button>
        </div>
        {#if smtpTestMessage}
            <p class="mt-2 text-sm text-success">{smtpTestMessage}</p>
        {/if}
        {#if smtpTestError}
            <p class="mt-2 text-sm text-destructive">{smtpTestError}</p>
        {/if}
        {#if !smtpEnabled}
            <p class="mt-2 text-xs text-muted-foreground">Enable SMTP above to send a test email.</p>
        {/if}
    </div>
</div>

<!-- Alert Rules -->
<div
    class="rounded-lg border bg-card p-4 sm:p-6 mb-6 transition-opacity duration-200 {alertLoaded ? 'opacity-100' : 'opacity-0'}"
>
    <h2 class="text-lg font-semibold text-foreground mb-1">Alert Rules</h2>
    <p class="text-sm text-muted-foreground mb-6">
        Global thresholds for email alerts. Alerts fire when a condition persists for the configured duration.
    </p>

    <div class="divide-y divide-border">
        {#each alertRules as rule (rule.metric_type)}
            <div class="py-4 first:pt-0 last:pb-0">
                <div class="flex items-start gap-4">
                    <!-- Toggle -->
                    <Toggle bind:checked={rule.enabled} class="mt-0.5" />

                    <!-- Label + inputs -->
                    <div class="flex-1 min-w-0">
                        <p class="text-sm font-medium text-foreground mb-3">
                            {ALERT_METRIC_LABELS[rule.metric_type]}
                        </p>

                        <div class="flex flex-wrap gap-4 {rule.enabled ? '' : 'opacity-40 pointer-events-none'}">
                            <!-- Threshold (hidden for server_down) -->
                            {#if rule.metric_type !== 'server_down'}
                                <div>
                                    <label class="block text-xs text-muted-foreground mb-1">Threshold</label>
                                    <div class="flex items-center gap-1.5">
                                        <input
                                            type="number"
                                            min="0"
                                            step={rule.metric_type === 'load_avg' ? '0.1' : '1'}
                                            bind:value={rule.threshold}
                                            placeholder={String(ALERT_DEFAULTS[rule.metric_type].threshold)}
                                            class="w-20 rounded-lg border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                                        />
                                        {#if thresholdUnit(rule.metric_type)}
                                            <span class="text-xs text-muted-foreground">{thresholdUnit(rule.metric_type)}</span>
                                        {/if}
                                    </div>
                                </div>
                            {/if}

                            <!-- Duration -->
                            <div>
                                <label class="block text-xs text-muted-foreground mb-1">Duration</label>
                                <div class="flex items-center gap-1.5">
                                    <input
                                        type="number"
                                        min="1"
                                        step="1"
                                        bind:value={rule.duration_minutes}
                                        placeholder={String(ALERT_DEFAULTS[rule.metric_type].duration)}
                                        class="w-16 rounded-lg border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                                    />
                                    <span class="text-xs text-muted-foreground">min</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        {/each}
    </div>

    {#if alertSaveError}
        <p class="mt-4 text-sm text-destructive">{alertSaveError}</p>
    {/if}

    <div class="mt-6">
        <button
            type="button"
            onclick={handleSaveAlerts}
            disabled={alertSaving || alertSaveSuccess}
            class="flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
            {#if alertSaveSuccess}
                <Check class="h-4 w-4" />
                Saved
            {:else}
                {alertSaving ? "Saving..." : "Save alert rules"}
            {/if}
        </button>
    </div>
</div>

<!-- Toast — API save errors -->
{#if toastMessage}
    <div
        transition:fly={{ y: 16, duration: 200 }}
        role="alert"
        class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 rounded-lg border border-destructive/40 bg-background px-4 py-3 text-sm text-destructive shadow-lg w-max max-w-[calc(100vw-2rem)]"
    >
        <TriangleAlert class="h-4 w-4 shrink-0" />
        <span>{toastMessage}</span>
        <button
            type="button"
            onclick={dismissToast}
            class="ml-1 rounded p-0.5 hover:bg-destructive/10 transition-colors"
            aria-label="Dismiss"
        >
            <X class="h-3.5 w-3.5" />
        </button>
    </div>
{/if}
