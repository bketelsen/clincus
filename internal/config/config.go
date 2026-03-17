package config

import (
	"os"
	"path/filepath"
)

// Config represents the complete configuration
type Config struct {
	Defaults  DefaultsConfig           `toml:"defaults" json:"defaults"`
	Paths     PathsConfig              `toml:"paths" json:"paths"`
	Incus     IncusConfig              `toml:"incus" json:"incus"`
	Tool      ToolConfig               `toml:"tool" json:"tool"`
	Mounts    MountsConfig             `toml:"mounts" json:"mounts"`
	Limits    LimitsConfig             `toml:"limits" json:"limits"`
	Git       GitConfig                `toml:"git" json:"git"`
	Security  SecurityConfig           `toml:"security" json:"security"`
	Profiles  map[string]ProfileConfig `toml:"profiles" json:"profiles"`
	Dashboard DashboardConfig          `toml:"dashboard" json:"dashboard"`
}

// GitConfig contains git-related security settings
type GitConfig struct {
	WritableHooks *bool `toml:"writable_hooks" json:"writable_hooks"` // Allow container to write to .git/hooks (default: false)
}

// SecurityConfig contains security-related settings for workspace protection
type SecurityConfig struct {
	// ProtectedPaths is a list of paths (relative to workspace) to mount read-only
	// These paths are protected to prevent containers from modifying files that could
	// execute automatically on the host (e.g., git hooks, IDE configs, etc.)
	// Defaults: [".git/hooks", ".git/config", ".husky", ".vscode"]
	ProtectedPaths []string `toml:"protected_paths" json:"protected_paths"`
	// AdditionalProtectedPaths allows adding more paths without replacing defaults
	AdditionalProtectedPaths []string `toml:"additional_protected_paths" json:"additional_protected_paths"`
	// DisableProtection completely disables read-only mounting of protected paths
	DisableProtection bool `toml:"disable_protection" json:"disable_protection"`
}

// GetEffectiveProtectedPaths returns the combined list of protected paths
func (s *SecurityConfig) GetEffectiveProtectedPaths() []string {
	if s.DisableProtection {
		return nil
	}
	paths := make([]string, 0, len(s.ProtectedPaths)+len(s.AdditionalProtectedPaths))
	paths = append(paths, s.ProtectedPaths...)
	paths = append(paths, s.AdditionalProtectedPaths...)
	return paths
}

// DefaultProtectedPaths returns the default list of paths to protect
func DefaultProtectedPaths() []string {
	return []string{
		".git/hooks",  // Git hooks - execute on git operations
		".git/config", // Git config - can set core.hooksPath to bypass hooks protection
		".husky",      // Husky - popular git hooks manager
		".vscode",     // VS Code - tasks.json can auto-execute, settings.json can inject shell args
	}
}

// DefaultsConfig contains default settings
type DefaultsConfig struct {
	Image      string `toml:"image" json:"image"`
	Persistent bool   `toml:"persistent" json:"persistent"`
	Model      string `toml:"model" json:"model"`
}

// PathsConfig contains path settings
type PathsConfig struct {
	SessionsDir           string `toml:"sessions_dir" json:"sessions_dir"`
	StorageDir            string `toml:"storage_dir" json:"storage_dir"`
	LogsDir               string `toml:"logs_dir" json:"logs_dir"`
	PreserveWorkspacePath bool   `toml:"preserve_workspace_path" json:"preserve_workspace_path"` // Mount workspace at same path as host (e.g., /home/user/project instead of /workspace)
}

// IncusConfig contains Incus-specific settings
type IncusConfig struct {
	Project      string `toml:"project" json:"project"`
	Group        string `toml:"group" json:"group"`
	CodeUID      int    `toml:"code_uid" json:"code_uid"`
	CodeUser     string `toml:"code_user" json:"code_user"`
	DisableShift bool   `toml:"disable_shift" json:"disable_shift"` // Disable UID shifting (for Colima/Lima environments)
}

// ProfileConfig represents a named profile
type ProfileConfig struct {
	Image       string            `toml:"image" json:"image"`
	Environment map[string]string `toml:"environment" json:"environment"`
	Persistent  bool              `toml:"persistent" json:"persistent"`
	Limits      *LimitsConfig     `toml:"limits" json:"limits,omitempty"`
}

// ToolConfig represents AI coding tool configuration
type ToolConfig struct {
	Name   string           `toml:"name" json:"name"`     // Tool name: "claude", "aider", "cursor", etc.
	Binary string           `toml:"binary" json:"binary"` // Binary name to execute (if empty, uses tool name)
	Claude ClaudeToolConfig `toml:"claude" json:"claude"`  // Claude-specific settings
}

// ClaudeToolConfig contains Claude Code-specific settings
type ClaudeToolConfig struct {
	EffortLevel string `toml:"effort_level" json:"effort_level"` // Effort level: "low", "medium", "high" (default: "medium")
}

// MountEntry represents a single directory mount configuration
type MountEntry struct {
	Host      string `toml:"host" json:"host"`           // Host path (supports ~ expansion)
	Container string `toml:"container" json:"container"` // Container path (must be absolute)
}

// MountsConfig contains mount-related configuration
type MountsConfig struct {
	Default []MountEntry `toml:"default" json:"default"` // Default mounts for all sessions
}

// LimitsConfig contains resource and time limits for containers
type LimitsConfig struct {
	CPU     CPULimits     `toml:"cpu" json:"cpu"`
	Memory  MemoryLimits  `toml:"memory" json:"memory"`
	Disk    DiskLimits    `toml:"disk" json:"disk"`
	Runtime RuntimeLimits `toml:"runtime" json:"runtime"`
}

// CPULimits contains CPU resource limits
type CPULimits struct {
	Count     string `toml:"count" json:"count"`         // "2", "0-3", "" (unlimited)
	Allowance string `toml:"allowance" json:"allowance"` // "50%", "25ms/100ms"
	Priority  int    `toml:"priority" json:"priority"`   // 0-10
}

// MemoryLimits contains memory resource limits
type MemoryLimits struct {
	Limit   string `toml:"limit" json:"limit"`     // "512MiB", "2GiB", "50%", "" (unlimited)
	Enforce string `toml:"enforce" json:"enforce"` // "hard" or "soft"
	Swap    string `toml:"swap" json:"swap"`       // "true", "false", or size
}

// DiskLimits contains disk I/O resource limits
type DiskLimits struct {
	Read      string `toml:"read" json:"read"`             // "10MiB/s", "1000iops", "" (unlimited)
	Write     string `toml:"write" json:"write"`           // "5MiB/s", "1000iops", "" (unlimited)
	Max       string `toml:"max" json:"max"`               // combined read+write limit
	Priority  int    `toml:"priority" json:"priority"`     // 0-10
	TmpfsSize string `toml:"tmpfs_size" json:"tmpfs_size"` // /tmp size: "2GiB", "1024MiB" (default: "2GiB")
}

// RuntimeLimits contains time-based and process limits
type RuntimeLimits struct {
	MaxDuration  string `toml:"max_duration" json:"max_duration"`   // "2h", "30m", "1h30m", "" (unlimited)
	MaxProcesses int    `toml:"max_processes" json:"max_processes"` // 0 = unlimited
	AutoStop     bool   `toml:"auto_stop" json:"auto_stop"`        // auto-stop when limit reached
	StopGraceful bool   `toml:"stop_graceful" json:"stop_graceful"` // graceful vs force stop
}

// DashboardConfig contains web dashboard settings
type DashboardConfig struct {
	Port           int      `toml:"port" json:"port"`
	WorkspaceRoots []string `toml:"workspace_roots" json:"workspace_roots"`
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp" // Fallback if home dir cannot be determined
	}
	baseDir := filepath.Join(homeDir, ".clincus")

	return &Config{
		Defaults: DefaultsConfig{
			Image:      "clincus",
			Persistent: false,
			Model:      "claude-sonnet-4-5",
		},
		Paths: PathsConfig{
			SessionsDir: filepath.Join(baseDir, "sessions"),
			StorageDir:  filepath.Join(baseDir, "storage"),
			LogsDir:     filepath.Join(baseDir, "logs"),
		},
		Incus: IncusConfig{
			Project:  "default",
			Group:    "incus-admin",
			CodeUID:  1000,
			CodeUser: "code",
		},
		Tool: ToolConfig{
			Name:   "claude",
			Binary: "", // Empty means use tool's default binary name
		},
		Mounts: MountsConfig{
			Default: []MountEntry{},
		},
		Git: GitConfig{
			WritableHooks: ptrBool(false),
		},
		Security: SecurityConfig{
			ProtectedPaths:           DefaultProtectedPaths(),
			AdditionalProtectedPaths: []string{},
			DisableProtection:        false,
		},
		Limits: LimitsConfig{
			CPU: CPULimits{
				Count:     "",
				Allowance: "",
				Priority:  0,
			},
			Memory: MemoryLimits{
				Limit:   "",
				Enforce: "soft",
				Swap:    "true",
			},
			Disk: DiskLimits{
				Read:      "",
				Write:     "",
				Max:       "",
				Priority:  0,
				TmpfsSize: "", // Default: use container root disk. Set "4GiB" etc. for RAM-backed tmpfs.
			},
			Runtime: RuntimeLimits{
				MaxDuration:  "",
				MaxProcesses: 0,
				AutoStop:     true,
				StopGraceful: true,
			},
		},
		Profiles:  make(map[string]ProfileConfig),
		Dashboard: DashboardConfig{Port: 3000},
	}
}

// GetConfigPaths returns the list of config file paths to check (in order)
// If CLINCUS_CONFIG environment variable is set, it is added as highest priority
func GetConfigPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	paths := []string{
		"/etc/clincus/config.toml",                            // System config
		filepath.Join(homeDir, ".config/clincus/config.toml"), // User config
		filepath.Join(workDir, ".clincus.toml"),               // Project config
	}

	// CLINCUS_CONFIG environment variable has highest priority
	if envConfig := os.Getenv("CLINCUS_CONFIG"); envConfig != "" {
		paths = append(paths, envConfig)
	}

	return paths
}

// ptrBool returns a pointer to a bool value
func ptrBool(b bool) *bool {
	return &b
}

// ExpandPath expands ~ in paths to home directory
func ExpandPath(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return path as-is if home dir cannot be determined
		}
		if len(path) == 1 {
			return homeDir
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// Merge merges another config into this one (other takes precedence)
func (c *Config) Merge(other *Config) {
	// Merge defaults
	if other.Defaults.Image != "" {
		c.Defaults.Image = other.Defaults.Image
	}
	if other.Defaults.Model != "" {
		c.Defaults.Model = other.Defaults.Model
	}
	// For booleans, we need a way to distinguish "not set" from "false"
	// In TOML, if a field is not present, it will be false (zero value)
	// This is a limitation - we'll just override if file exists
	c.Defaults.Persistent = other.Defaults.Persistent

	// Merge paths
	if other.Paths.SessionsDir != "" {
		c.Paths.SessionsDir = ExpandPath(other.Paths.SessionsDir)
	}
	if other.Paths.StorageDir != "" {
		c.Paths.StorageDir = ExpandPath(other.Paths.StorageDir)
	}
	if other.Paths.LogsDir != "" {
		c.Paths.LogsDir = ExpandPath(other.Paths.LogsDir)
	}
	if other.Paths.PreserveWorkspacePath {
		c.Paths.PreserveWorkspacePath = true
	}

	// Merge Incus settings
	if other.Incus.Project != "" {
		c.Incus.Project = other.Incus.Project
	}
	if other.Incus.Group != "" {
		c.Incus.Group = other.Incus.Group
	}
	if other.Incus.CodeUID != 0 {
		c.Incus.CodeUID = other.Incus.CodeUID
	}
	if other.Incus.CodeUser != "" {
		c.Incus.CodeUser = other.Incus.CodeUser
	}

	// Merge Tool settings
	if other.Tool.Name != "" {
		c.Tool.Name = other.Tool.Name
	}
	if other.Tool.Binary != "" {
		c.Tool.Binary = other.Tool.Binary
	}
	// Merge Claude-specific settings
	if other.Tool.Claude.EffortLevel != "" {
		c.Tool.Claude.EffortLevel = other.Tool.Claude.EffortLevel
	}
	// For DisableShift, if the other config sets it to true, use it
	if other.Incus.DisableShift {
		c.Incus.DisableShift = true
	}

	// Merge mounts - append from other config
	if len(other.Mounts.Default) > 0 {
		c.Mounts.Default = append(c.Mounts.Default, other.Mounts.Default...)
	}

	// Merge limits
	mergeLimits(&c.Limits, &other.Limits)

	// Merge git settings
	// Only override if explicitly set in the other config (nil means not set)
	if other.Git.WritableHooks != nil {
		c.Git.WritableHooks = other.Git.WritableHooks
	}

	// Merge security settings
	if len(other.Security.ProtectedPaths) > 0 {
		c.Security.ProtectedPaths = other.Security.ProtectedPaths
	}
	if len(other.Security.AdditionalProtectedPaths) > 0 {
		c.Security.AdditionalProtectedPaths = append(c.Security.AdditionalProtectedPaths, other.Security.AdditionalProtectedPaths...)
	}
	if other.Security.DisableProtection {
		c.Security.DisableProtection = true
	}

	// Merge profiles
	for name, profile := range other.Profiles {
		c.Profiles[name] = profile
	}

	// Merge dashboard
	if other.Dashboard.Port != 0 {
		c.Dashboard.Port = other.Dashboard.Port
	}
	if len(other.Dashboard.WorkspaceRoots) > 0 {
		c.Dashboard.WorkspaceRoots = other.Dashboard.WorkspaceRoots
	}
}

// mergeLimits merges limit configurations (other takes precedence)
func mergeLimits(base *LimitsConfig, other *LimitsConfig) {
	// Merge CPU limits
	if other.CPU.Count != "" {
		base.CPU.Count = other.CPU.Count
	}
	if other.CPU.Allowance != "" {
		base.CPU.Allowance = other.CPU.Allowance
	}
	if other.CPU.Priority != 0 {
		base.CPU.Priority = other.CPU.Priority
	}

	// Merge memory limits
	if other.Memory.Limit != "" {
		base.Memory.Limit = other.Memory.Limit
	}
	if other.Memory.Enforce != "" {
		base.Memory.Enforce = other.Memory.Enforce
	}
	if other.Memory.Swap != "" {
		base.Memory.Swap = other.Memory.Swap
	}

	// Merge disk limits
	if other.Disk.Read != "" {
		base.Disk.Read = other.Disk.Read
	}
	if other.Disk.Write != "" {
		base.Disk.Write = other.Disk.Write
	}
	if other.Disk.Max != "" {
		base.Disk.Max = other.Disk.Max
	}
	if other.Disk.Priority != 0 {
		base.Disk.Priority = other.Disk.Priority
	}
	if other.Disk.TmpfsSize != "" {
		base.Disk.TmpfsSize = other.Disk.TmpfsSize
	}

	// Merge runtime limits
	if other.Runtime.MaxDuration != "" {
		base.Runtime.MaxDuration = other.Runtime.MaxDuration
	}
	if other.Runtime.MaxProcesses != 0 {
		base.Runtime.MaxProcesses = other.Runtime.MaxProcesses
	}
	// For booleans, we take the other value if it differs from default
	// This is imperfect but works for most cases
	base.Runtime.AutoStop = other.Runtime.AutoStop
	base.Runtime.StopGraceful = other.Runtime.StopGraceful
}

// GetProfile returns a profile by name, or nil if not found
func (c *Config) GetProfile(name string) *ProfileConfig {
	if profile, ok := c.Profiles[name]; ok {
		return &profile
	}
	return nil
}

// ApplyProfile applies a profile's settings to the defaults
func (c *Config) ApplyProfile(name string) bool {
	profile := c.GetProfile(name)
	if profile == nil {
		return false
	}

	if profile.Image != "" {
		c.Defaults.Image = profile.Image
	}
	c.Defaults.Persistent = profile.Persistent

	// Apply profile limits if present
	if profile.Limits != nil {
		mergeLimits(&c.Limits, profile.Limits)
	}

	return true
}
