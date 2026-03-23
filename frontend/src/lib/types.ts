/**
 * TypeScript type definitions for Watchflare Frontend
 */

// ===== User & Authentication =====

export type Theme = 'light' | 'dark' | 'system';
export type TimeFormat = '24h' | '12h';
export type TemperatureUnit = 'celsius' | 'fahrenheit';
export type NetworkUnit = 'bytes' | 'bits';
export type DiskUnit = 'bytes' | 'bits';

export interface User {
	id: number;
	email: string;
	username: string;
	default_time_range: TimeRange;
	theme: Theme;
	time_format: TimeFormat;
	temperature_unit: TemperatureUnit;
	network_unit: NetworkUnit;
	disk_unit: DiskUnit;
	gauge_warning_threshold: number;
	gauge_critical_threshold: number;
	created_at: string;
}

export interface LoginRequest {
	email: string;
	password: string;
}

export interface RegisterRequest {
	email: string;
	password: string;
}

export interface ChangePasswordRequest {
	current_password: string;
	new_password: string;
}

// ===== Server =====

export type ServerStatus = 'online' | 'offline' | 'pending' | 'paused' | 'ip_mismatch';
export type EnvironmentType = 'physical' | 'physical_with_containers' | 'vm' | 'vm_with_containers';

export interface Server {
	id: string;
	name: string;
	hostname: string;
	platform: string | null;
	platform_version: string | null;
	platform_family: string | null;
	architecture: string | null;
	kernel: string | null;
	ip_address_v4: string;
	ip_address_v6: string | null;
	configured_ip: string;
	ignore_ip_mismatch: boolean;
	status: ServerStatus;
	last_seen: string;
	created_at: string;
	environment_type: EnvironmentType;
	container_runtime: string | null;
	hypervisor: string | null;
	reactivated_at: string | null;
	agent_version: string | null;
	server_uuid: string;
}

export interface ServerWithMetrics {
	server: Server;
	latestMetric?: Metric;
}

export interface CreateServerRequest {
	name: string;
	configured_ip?: string;
	allow_any_ip?: boolean;
}

export interface UpdateConfiguredIPRequest {
	configured_ip: string;
}

// ===== Metrics =====

export type TimeRange = '1h' | '12h' | '24h' | '7d' | '30d';

export interface SensorReading {
	key: string;
	temperature_celsius: number;
}

export interface Metric {
	id: number;
	server_id: string;
	timestamp: string;
	cpu_usage_percent: number;
	memory_used_bytes: number;
	memory_total_bytes: number;
	memory_available_bytes: number;
	disk_used_bytes: number;
	disk_total_bytes: number;
	load_avg_1min: number;
	load_avg_5min: number;
	load_avg_15min: number;
	uptime_seconds: number;
	disk_read_bytes_per_sec: number;
	disk_write_bytes_per_sec: number;
	network_rx_bytes_per_sec: number;
	network_tx_bytes_per_sec: number;
	cpu_temperature_celsius: number;
	sensor_readings?: SensorReading[];
}

export interface AggregatedMetric {
	timestamp: string;
	cpu_usage_percent: number;
	memory_used_bytes: number;
	memory_total_bytes: number;
	memory_available_bytes: number;
	disk_used_bytes: number;
	disk_total_bytes: number;
	load_avg_1min: number;
	load_avg_5min: number;
	load_avg_15min: number;
	disk_read_bytes_per_sec: number;
	disk_write_bytes_per_sec: number;
	network_rx_bytes_per_sec: number;
	network_tx_bytes_per_sec: number;
	cpu_temperature_celsius: number;
	server_count: number;
}

export interface ContainerMetric {
	id: string;
	server_id: string;
	timestamp: string;
	container_id: string;
	container_name: string;
	image: string;
	cpu_percent: number;
	memory_used_bytes: number;
	memory_limit_bytes: number;
	network_rx_bytes_per_sec: number;
	network_tx_bytes_per_sec: number;
}

export interface MetricsQueryParams {
	time_range?: TimeRange;
	limit?: number;
	offset?: number;
}

// ===== Dropped Metrics =====

export interface DroppedMetric {
	hostname: string;
	total_dropped: number;
	first_dropped_at: string;
	last_dropped_at: string;
	downtime_duration: number;
}

// ===== Packages =====

export interface Package {
	id: number;
	server_id: string;
	name: string;
	version: string;
	architecture: string;
	package_manager: string;
	source: string;
	installed_at: string | null;
	package_size: number;
	description: string;
	first_seen: string;
	last_seen: string;
}

export interface PackageManagerStat {
	package_manager: string;
	count: number;
}

export interface PackageStats {
	total_packages: number;
	recent_changes?: number;
	by_package_manager: PackageManagerStat[];
}

export interface PackageCollection {
	id: number;
	server_id: string;
	timestamp: string;
	collection_type: string;
	package_count: number;
	changes_count: number;
	duration_ms: number;
	status: string;
	error_message: string;
}

export interface PackageHistory {
	id: number;
	server_id: string;
	timestamp: string;
	name: string;
	version: string;
	architecture: string;
	package_manager: string;
	source: string;
	package_size: number;
	description: string;
	change_type: 'added' | 'removed' | 'updated' | 'initial';
}

// ===== SSE Events =====

export type SSEEventType = 'connected' | 'server_update' | 'metrics_update' | 'aggregated_metrics_update' | 'container_metrics_update';

export interface SSEEvent {
	type: SSEEventType;
	data: unknown;
}

export interface ServerUpdateEvent {
	id: string;
	status: ServerStatus;
	last_seen: string;
	ip_address_v4?: string;
	ip_address_v6?: string;
	configured_ip?: string;
	ignore_ip_mismatch?: boolean;
	reactivated?: boolean;
	hostname?: string;
}

export interface MetricsUpdateEvent extends Metric {
	server_id: string;
}

export interface AggregatedMetricsUpdateEvent extends AggregatedMetric {
	// Same as AggregatedMetric
}

// ===== API Responses =====

export interface APIResponse<T> {
	success?: boolean;
	data?: T;
	error?: string;
	message?: string;
}

export interface LoginResponse {
	message: string;
	user: User;
}

export interface RegisterResponse {
	message: string;
	user: User;
}

export interface CreateServerResponse {
	message: string;
	server: Server;
	token: string;
	agent_key: string;
	backend_host: string;
}

export interface RegenerateTokenResponse {
	message: string;
	token: string;
}

export interface GetServerResponse {
	server: Server;
	clock_desync: boolean;
}

export interface ListServersResponse {
	servers: Server[];
	total: number;
	page: number;
	per_page: number;
}

export interface GetMetricsResponse {
	metrics: Metric[];
}

export interface GetAggregatedMetricsResponse {
	metrics: AggregatedMetric[];
}

export interface GetDroppedMetricsResponse {
	dropped_metrics: DroppedMetric[];
}

export interface GetContainerMetricsResponse {
	metrics: ContainerMetric[];
}

export interface GetPackagesResponse {
	packages: Package[];
	total_count: number;
	limit: number;
	offset: number;
}

export interface GetPackageStatsResponse extends PackageStats {
	last_collection: PackageCollection | null;
}

export interface GetPackageCollectionsResponse {
	collections: PackageCollection[];
	total_count: number;
	limit: number;
	offset: number;
}

export interface GetPackageHistoryResponse {
	history: PackageHistory[];
	total_count: number;
	limit: number;
	offset: number;
}

export interface CurrentUserResponse {
	user: User;
}

// ===== Toast Notifications =====

export type ToastType = 'info' | 'success' | 'warning' | 'error';

export interface Toast {
	id: number;
	message: string;
	type: ToastType;
}

export interface ToastStore {
	subscribe: (fn: (toasts: Toast[]) => void) => () => void;
	add: (message: string, type?: ToastType, duration?: number) => number;
	remove: (id: number) => void;
	clear: () => void;
}

// ===== Component Props =====

export interface ChartProps {
	data: Metric[] | AggregatedMetric[];
}

export interface ServerTableProps {
	servers: ServerWithMetrics[];
	metricsData: Record<string, Metric[]>;
}
