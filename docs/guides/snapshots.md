# Snapshots

Snapshots capture the complete state of a container — all files, installed packages, and
(optionally) running process memory. They allow you to checkpoint a working state before
an experiment and roll back if things go wrong.

---

## Overview

Snapshots are an Incus-native feature. Clincus wraps the Incus snapshot API with convenient
auto-detection of the container for the current workspace.

Use cases:

- **Safe experimentation** — snapshot before letting the AI try a risky refactor
- **Branch points** — create snapshots at multiple decision points and restore to try
  different approaches
- **Recovery** — restore from a known-good state if the AI tool modifies things
  unexpectedly

!!! warning "Container must be stopped to restore"
    Restoring a snapshot replaces the container filesystem. The container must be stopped
    first. Use `clincus kill` or stop it from inside before restoring.

---

## Creating Snapshots

```bash
# Auto-named snapshot (snap-YYYYMMDD-HHMMSS)
clincus snapshot create

# Named snapshot
clincus snapshot create checkpoint-1

# Stateful snapshot (includes process memory — requires CRIU)
clincus snapshot create --stateful live-session
```

### Target a Specific Container

By default, Clincus resolves the container for the current workspace. Override with:

```bash
clincus snapshot create -c clincus-a1b2c3-1 my-snapshot
```

You can also set `CLINCUS_CONTAINER` in your environment to pin the target container.

---

## Listing Snapshots

```bash
# Snapshots for the current workspace's container
clincus snapshot list

# All Clincus containers
clincus snapshot list --all

# JSON output
clincus snapshot list --format json
```

Example output:

```
Snapshots for clincus-a1b2c3-1:

NAME                CREATED                  STATEFUL
checkpoint-1        2026-01-01 12:00:00      no
snap-20260101       2026-01-01 11:30:00      no

Total: 2 snapshots
```

---

## Restoring a Snapshot

```bash
# With confirmation prompt
clincus snapshot restore checkpoint-1

# Skip confirmation
clincus snapshot restore checkpoint-1 -f
```

Clincus verifies the container is stopped before restoring. If it is still running, it
reports an error with instructions:

```
Error: container 'clincus-a1b2c3-1' must be stopped before restore
(use 'clincus container stop clincus-a1b2c3-1')
```

After restoring, start a new session from the same workspace as usual:

```bash
clincus shell ~/my-project
```

---

## Snapshot Details

```bash
clincus snapshot info checkpoint-1
clincus snapshot info checkpoint-1 --format json
```

Output:

```
Snapshot: checkpoint-1
Container: clincus-a1b2c3-1
Created: 2026-01-01 12:00:00
Stateful: no
```

---

## Deleting Snapshots

```bash
# Delete a specific snapshot
clincus snapshot delete checkpoint-1

# Delete all snapshots for the current workspace's container (with confirmation)
clincus snapshot delete --all

# Skip confirmation
clincus snapshot delete --all -f
```

Snapshots consume disk space in the Incus storage pool. Clean them up regularly to free space.

---

## Stateful Snapshots

Stateful snapshots preserve the memory state of all running processes in addition to the
filesystem. This allows restoring a container to exactly the same point in execution, with
all processes resumed.

Stateful snapshots require [CRIU](https://criu.org/) to be installed on the host.

```bash
clincus snapshot create --stateful my-checkpoint
clincus snapshot restore my-checkpoint --stateful
```

!!! note "Stateful snapshots and AI tools"
    Stateful snapshots of a running AI tool session can be useful for creating an exact
    restore point mid-conversation, but the tool must support being resumed from a
    memory snapshot (most do, since they are just processes). Test this with your specific
    tool before relying on it.
