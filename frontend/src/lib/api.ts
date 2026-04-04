import type {
  LoginResponse,
  RegisterResponse,
  CreateServerResponse,
  RegenerateTokenResponse,
  GetServerResponse,
  ListServersResponse,
  GetMetricsResponse,
  GetAggregatedMetricsResponse,
  GetDroppedMetricsResponse,
  GetContainerMetricsResponse,
  GetSensorReadingsResponse,
  GetPackagesResponse,
  GetPackageStatsResponse,
  GetPackageCollectionsResponse,
  GetPackageHistoryResponse,
  CurrentUserResponse,
  MetricsQueryParams,
  User,
  GetSMTPSettingsResponse,
  UpdateSMTPSettingsRequest,
} from "./types";

export const API_BASE_URL = import.meta.env.VITE_API_URL || "/api";

// Build query string from params, filtering out undefined/null/empty values
export function buildQueryString(
  params: Record<string, string | number | boolean | undefined | null>,
): string {
  const qs = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== "") {
      qs.append(key, String(value));
    }
  }
  const str = qs.toString();
  return str ? "?" + str : "";
}

interface ApiRequestOptions extends RequestInit {
  headers?: Record<string, string>;
}

// Custom API error class
export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public data?: { error?: string; message?: string },
  ) {
    super(data?.error || data?.message || statusText || "API request failed");
    this.name = "ApiError";
  }

  get isAuthError(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }

  get isNotFound(): boolean {
    return this.status === 404;
  }

  get isServerError(): boolean {
    return this.status >= 500;
  }
}

// Check if initial setup is required (no users exist)
export async function checkSetupRequired(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/setup-required`, {
      credentials: "include",
    });
    const data = await response.json();
    return data.setup_required;
  } catch (err) {
    if (import.meta.env.DEV)
      console.error("Failed to check setup status:", err);
    return false;
  }
}

// Handle authentication errors
async function handleAuthError(): Promise<never> {
  try {
    const setupRequired = await checkSetupRequired();
    if (setupRequired) {
      // No users exist, redirect to registration
      window.location.href = "/register";
      throw new ApiError(401, "Unauthorized", {
        message: "Redirecting to registration",
      });
    } else {
      // Users exist but not authenticated, redirect to login
      window.location.href = "/login";
      throw new ApiError(401, "Unauthorized", {
        message: "Redirecting to login",
      });
    }
  } catch (err) {
    // If checking setup status fails, default to login
    window.location.href = "/login";
    throw new ApiError(401, "Unauthorized", {
      message: "Redirecting to login",
    });
  }
}

// Make API request with credentials (cookies sent automatically)
async function apiRequest<T>(
  endpoint: string,
  options: ApiRequestOptions = {},
  skipAuthRedirect = false,
): Promise<T> {
  const headers = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  let response: Response;
  let data: unknown;

  try {
    response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
      credentials: "include", // Important: send cookies with requests
    });

    // Try to parse JSON response
    try {
      data = await response.json();
    } catch (parseError) {
      // If JSON parsing fails, create error with status text
      if (!response.ok) {
        throw new ApiError(response.status, response.statusText);
      }
      throw new ApiError(500, "Invalid response format");
    }
  } catch (err) {
    // Network or fetch errors
    if (err instanceof ApiError) {
      throw err;
    }
    // Network error (e.g., no internet, CORS, etc.)
    throw new ApiError(0, "Network error", {
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    });
  }

  if (!response.ok) {
    // Handle authentication errors (skip for auth endpoints like login/register)
    if (response.status === 401 && !skipAuthRedirect) {
      await handleAuthError();
    }

    // Throw API error for other cases
    throw new ApiError(
      response.status,
      response.statusText,
      data as { error?: string; message?: string },
    );
  }

  return data as T;
}

// Auth API calls
export async function register(
  email: string,
  password: string,
  username?: string,
): Promise<RegisterResponse> {
  return apiRequest<RegisterResponse>(
    "/auth/register",
    {
      method: "POST",
      body: JSON.stringify({ email, password, username: username || "" }),
    },
    true,
  );
}

export async function login(
  email: string,
  password: string,
): Promise<LoginResponse> {
  return apiRequest<LoginResponse>(
    "/auth/login",
    {
      method: "POST",
      body: JSON.stringify({ email, password }),
    },
    true,
  );
}

export async function logout(): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/auth/logout", {
    method: "POST",
  });
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/auth/change-password", {
    method: "PUT",
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
}

export async function changeEmail(
  newEmail: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/auth/change-email", {
    method: "PUT",
    body: JSON.stringify({
      new_email: newEmail,
    }),
  });
}

export async function changeUsername(
  username: string,
): Promise<{ message: string; user: User }> {
  return apiRequest<{ message: string; user: User }>("/auth/change-username", {
    method: "PUT",
    body: JSON.stringify({ username }),
  });
}

// Server API calls
export async function listServers(params?: {
  page?: number;
  perPage?: number;
  sort?: string;
  order?: "asc" | "desc";
  status?: string;
  search?: string;
}): Promise<ListServersResponse> {
  const query = buildQueryString({
    page: params?.page,
    per_page: params?.perPage,
    sort: params?.sort,
    order: params?.order,
    status: params?.status,
    search: params?.search,
  });
  return apiRequest<ListServersResponse>(`/servers${query}`);
}

export async function getServer(id: string): Promise<GetServerResponse> {
  return apiRequest<GetServerResponse>(`/servers/${id}`);
}

export async function createServer(
  name: string,
  configuredIP?: string,
  allowAnyIP?: boolean,
): Promise<CreateServerResponse> {
  return apiRequest<CreateServerResponse>("/servers", {
    method: "POST",
    body: JSON.stringify({
      name,
      configured_ip: configuredIP,
      allow_any_ip: allowAnyIP,
    }),
  });
}

export async function pauseServer(id: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}/pause`, {
    method: "PUT",
  });
}

export async function resumeServer(id: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}/resume`, {
    method: "PUT",
  });
}

export async function deleteServer(id: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}`, {
    method: "DELETE",
  });
}

export async function regenerateToken(
  id: string,
): Promise<RegenerateTokenResponse> {
  return apiRequest<RegenerateTokenResponse>(
    `/servers/${id}/regenerate-token`,
    {
      method: "POST",
    },
  );
}

export async function validateIP(
  id: string,
  selectedIP: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}/validate-ip`, {
    method: "PUT",
    body: JSON.stringify({ selected_ip: selectedIP }),
  });
}

export async function renameServer(
  id: string,
  newName: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}/rename`, {
    method: "PUT",
    body: JSON.stringify({ new_name: newName }),
  });
}

export async function updateConfiguredIP(
  id: string,
  newIP: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}/change-ip`, {
    method: "PUT",
    body: JSON.stringify({ new_ip: newIP }),
  });
}

export async function ignoreIPMismatch(
  id: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(`/servers/${id}/ignore-ip-mismatch`, {
    method: "PUT",
  });
}

export async function dismissReactivation(
  id: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(
    `/servers/${id}/dismiss-reactivation`,
    {
      method: "PUT",
    },
  );
}

// User preferences API calls
export async function getCurrentUser(): Promise<CurrentUserResponse> {
  return apiRequest<CurrentUserResponse>("/auth/user");
}

export interface UpdatePreferencesPayload {
  default_time_range?: string;
  theme?: string;
  time_format?: string;
  temperature_unit?: string;
  network_unit?: string;
  disk_unit?: string;
  gauge_warning_threshold?: number;
  gauge_critical_threshold?: number;
}

export async function updatePreferences(
  payload: UpdatePreferencesPayload,
): Promise<{ message: string; user: User }> {
  return apiRequest<{ message: string; user: User }>("/auth/preferences", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

// Metrics API calls
export async function getServerMetrics(
  serverId: string,
  params: MetricsQueryParams = {},
): Promise<GetMetricsResponse> {
  const query = buildQueryString({
    time_range: params.time_range,
    limit: params.limit,
    offset: params.offset,
  });
  return apiRequest<GetMetricsResponse>(`/servers/${serverId}/metrics${query}`);
}

// Get dropped metrics summary for the last 24 hours
export async function getDroppedMetrics(): Promise<GetDroppedMetricsResponse> {
  return apiRequest<GetDroppedMetricsResponse>("/servers/dropped-metrics");
}

// Get per-sensor temperature readings for a specific server
export async function getSensorReadings(
  serverId: string,
  timeRange?: string,
): Promise<GetSensorReadingsResponse> {
  const query = buildQueryString({ time_range: timeRange });
  return apiRequest<GetSensorReadingsResponse>(
    `/servers/${serverId}/sensor-readings${query}`,
  );
}

// Get container metrics for a specific server
export async function getContainerMetrics(
  serverId: string,
  timeRange?: string,
): Promise<GetContainerMetricsResponse> {
  const query = buildQueryString({ time_range: timeRange });
  return apiRequest<GetContainerMetricsResponse>(
    `/servers/${serverId}/container-metrics${query}`,
  );
}

// Get aggregated metrics from all online servers
export async function getAggregatedMetrics(
  timeRange?: string,
): Promise<GetAggregatedMetricsResponse> {
  const query = buildQueryString({ time_range: timeRange });
  return apiRequest<GetAggregatedMetricsResponse>(
    `/servers/metrics/aggregated${query}`,
  );
}

// Package API calls
interface PackageQueryParams {
  limit?: number;
  offset?: number;
  package_manager?: string;
  search?: string;
}

export async function getServerPackages(
  serverId: string,
  params: PackageQueryParams = {},
): Promise<GetPackagesResponse> {
  const query = buildQueryString({
    limit: params.limit,
    offset: params.offset,
    package_manager: params.package_manager,
    search: params.search,
  });
  return apiRequest<GetPackagesResponse>(
    `/servers/${serverId}/packages${query}`,
  );
}

export async function getPackageStats(
  serverId: string,
): Promise<GetPackageStatsResponse> {
  return apiRequest<GetPackageStatsResponse>(
    `/servers/${serverId}/packages/stats`,
  );
}

interface CollectionQueryParams {
  limit?: number;
  offset?: number;
}

export async function getPackageCollections(
  serverId: string,
  params: CollectionQueryParams = {},
): Promise<GetPackageCollectionsResponse> {
  const query = buildQueryString({
    limit: params.limit,
    offset: params.offset,
  });
  return apiRequest<GetPackageCollectionsResponse>(
    `/servers/${serverId}/packages/collections${query}`,
  );
}

interface HistoryQueryParams extends CollectionQueryParams {
  exclude_initial?: boolean;
}

export async function getPackageHistory(
  serverId: string,
  params: HistoryQueryParams = {},
): Promise<GetPackageHistoryResponse> {
  const query = buildQueryString({
    limit: params.limit,
    offset: params.offset,
    exclude_initial: params.exclude_initial,
  });
  return apiRequest<GetPackageHistoryResponse>(
    `/servers/${serverId}/packages/history${query}`,
  );
}

export async function getLatestAgentVersion(): Promise<{
  latest_version: string;
}> {
  return apiRequest<{ latest_version: string }>("/agent/latest-version");
}

// Settings API calls
export async function getSmtpSettings(): Promise<GetSMTPSettingsResponse> {
  return apiRequest<GetSMTPSettingsResponse>("/settings/smtp");
}

export async function updateSmtpSettings(
  data: UpdateSMTPSettingsRequest,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/settings/smtp", {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function testSmtpConnection(
  recipient?: string,
): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/settings/smtp/test", {
    method: "POST",
    body: JSON.stringify({ recipient: recipient ?? "" }),
  });
}
