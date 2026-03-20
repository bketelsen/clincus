export interface Session {
  id: string;
  workspace: string;
  tool: string;
  slot: number;
  status: string;
  persistent: boolean;
}

export interface Workspace {
  path: string;
  name: string;
  root: string;
  has_config: boolean;
  active_sessions: number;
}

export interface HistoryEntry {
  id: string;
  workspace: string;
  tool: string;
  started: string;
  stopped: string;
  persistent: boolean;
  exit_code: number;
}

export interface WSMessage {
  type: string;
  data?: string;
  cols?: number;
  rows?: number;
  code?: number;
  message?: string;
}

// Full configuration types matching the Go config.Config struct (AC4)

export interface DefaultsConfig {
  image: string;
  persistent: boolean;
  model: string;
}

export interface PathsConfig {
  sessions_dir: string;
  storage_dir: string;
  logs_dir: string;
  preserve_workspace_path: boolean;
}

export interface IncusConfig {
  project: string;
  group: string;
  code_uid: number;
  code_user: string;
  disable_shift: boolean;
}

export interface ClaudeToolConfig {
  effort_level: string;
}

export interface ToolConfig {
  name: string;
  binary: string;
  claude: ClaudeToolConfig;
}

export interface MountEntry {
  host: string;
  container: string;
}

export interface MountsConfig {
  default: MountEntry[];
}

export interface CPULimits {
  count: string;
  allowance: string;
  priority: number;
}

export interface MemoryLimits {
  limit: string;
  enforce: string;
  swap: string;
}

export interface DiskLimits {
  read: string;
  write: string;
  max: string;
  priority: number;
  tmpfs_size: string;
}

export interface RuntimeLimits {
  max_duration: string;
  max_processes: number;
  auto_stop: boolean;
  stop_graceful: boolean;
}

export interface LimitsConfig {
  cpu: CPULimits;
  memory: MemoryLimits;
  disk: DiskLimits;
  runtime: RuntimeLimits;
}

export interface GitConfig {
  writable_hooks: boolean | null;
}

export interface SecurityConfig {
  protected_paths: string[];
  additional_protected_paths: string[];
  disable_protection: boolean;
}

export interface ProfileConfig {
  image: string;
  environment: Record<string, string>;
  persistent: boolean;
  limits?: LimitsConfig;
}

export interface DashboardConfig {
  port: number;
  workspace_roots: string[];
}

export interface ClincusConfig {
  defaults: DefaultsConfig;
  paths: PathsConfig;
  incus: IncusConfig;
  tool: ToolConfig;
  mounts: MountsConfig;
  limits: LimitsConfig;
  git: GitConfig;
  security: SecurityConfig;
  profiles: Record<string, ProfileConfig>;
  dashboard: DashboardConfig;
}

export interface FileEntry {
  name: string;
  type: 'file' | 'dir' | 'symlink';
  size: number;
}

export interface FileListResponse {
  path: string;
  entries: FileEntry[];
}

export interface FileContentResponse {
  path: string;
  content: string;
  size: number;
}
