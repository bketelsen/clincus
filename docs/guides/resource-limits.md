# Resource Limits

Clincus lets you constrain the CPU, memory, disk I/O, and runtime of each session. Limits
can be set via CLI flags for a single session, in a project config (`.clincus.toml`), or in
the user config (`~/.config/clincus/config.toml`).

CLI flags take precedence over config file values.

---

## CPU Limits

Control how much CPU the container can use.

| Flag | Config key | Format | Example |
|------|-----------|--------|---------|
| `--limit-cpu` | `[limits.cpu] count` | count or range | `"2"`, `"0-3"`, `"0,1,3"` |
| `--limit-cpu-allowance` | `[limits.cpu] allowance` | percentage or period | `"50%"`, `"25ms/100ms"` |
| `--limit-cpu-priority` | `[limits.cpu] priority` | integer 0–10 | `5` |

```bash
# Limit to 2 CPU cores
clincus shell --limit-cpu 2

# Limit to 50% of one CPU
clincus shell --limit-cpu-allowance 50%

# Use cores 0 and 1 only
clincus shell --limit-cpu "0,1"
```

Config file:

```toml
[limits.cpu]
count = "2"
allowance = "50%"
priority = 5
```

---

## Memory Limits

| Flag | Config key | Format | Example |
|------|-----------|--------|---------|
| `--limit-memory` | `[limits.memory] limit` | size or percentage | `"2GiB"`, `"512MiB"`, `"50%"` |
| `--limit-memory-swap` | `[limits.memory] swap` | `"true"`, `"false"`, or size | `"false"` |
| `--limit-memory-enforce` | `[limits.memory] enforce` | `"hard"` or `"soft"` | `"hard"` |

```bash
# Limit to 4 GiB RAM
clincus shell --limit-memory 4GiB

# Limit to 4 GiB with hard enforcement (OOM kill on exceed)
clincus shell --limit-memory 4GiB --limit-memory-enforce hard

# Disable swap
clincus shell --limit-memory-swap false
```

Config file:

```toml
[limits.memory]
limit = "4GiB"
enforce = "hard"
swap = "false"
```

**Enforcement modes:**

- `soft` (default) — the kernel tries to honor the limit but will allow bursts
- `hard` — strictly enforced; processes are OOM-killed if they exceed the limit

---

## Disk I/O Limits

| Flag | Config key | Format | Example |
|------|-----------|--------|---------|
| `--limit-disk-read` | `[limits.disk] read` | rate or IOPS | `"10MiB/s"`, `"1000iops"` |
| `--limit-disk-write` | `[limits.disk] write` | rate or IOPS | `"5MiB/s"` |
| `--limit-disk-max` | `[limits.disk] max` | combined rate | `"20MiB/s"` |
| `--limit-disk-priority` | `[limits.disk] priority` | integer 0–10 | `5` |

```bash
# Limit disk read to 50 MiB/s and write to 25 MiB/s
clincus shell --limit-disk-read 50MiB/s --limit-disk-write 25MiB/s
```

Config file:

```toml
[limits.disk]
read = "50MiB/s"
write = "25MiB/s"
priority = 3
```

### tmpfs Size

The container `/tmp` directory is backed by a tmpfs. Control its size:

```toml
[limits.disk]
tmpfs_size = "4GiB"
```

The default is to use the container root disk (no separate tmpfs).

---

## Runtime Limits

| Flag | Config key | Format | Example |
|------|-----------|--------|---------|
| `--limit-processes` | `[limits.runtime] max_processes` | integer (0 = unlimited) | `500` |
| `--limit-duration` | `[limits.runtime] max_duration` | duration string | `"2h"`, `"30m"`, `"1h30m"` |

```bash
# Kill the container after 2 hours
clincus shell --limit-duration 2h

# Limit to 500 processes
clincus shell --limit-processes 500
```

Config file:

```toml
[limits.runtime]
max_duration = "2h"
max_processes = 500
auto_stop = true
stop_graceful = true
```

**`auto_stop`** — when `true`, the container is automatically stopped when `max_duration` is
reached (default: `true`).

**`stop_graceful`** — when `true`, a graceful shutdown signal is sent first before force-kill
(default: `true`).

---

## Named Profiles with Limits

Define reusable profiles that bundle a set of limits:

```toml
# ~/.config/clincus/config.toml

[profiles.restricted]
[profiles.restricted.limits.cpu]
count = "1"
allowance = "50%"

[profiles.restricted.limits.memory]
limit = "1GiB"
enforce = "hard"

[profiles.restricted.limits.runtime]
max_duration = "30m"
auto_stop = true
```

Apply:

```bash
clincus shell --profile restricted ~/untrusted-project
```

---

## Combining Config and CLI

Config file limits are loaded first; any CLI flags you specify override them for that session:

```bash
# Config sets memory=2GiB; this session overrides it to 8GiB
clincus shell --limit-memory 8GiB
```
