# Images

Clincus sessions run inside Incus container images. The default `clincus` image includes
everything needed for AI-assisted coding. You can also build custom images for specialized
environments.

---

## The Default Image

The built-in `clincus` image includes:

- Ubuntu 24.04 base
- Node.js LTS (via nvm)
- Claude Code CLI (`claude`)
- Docker CLI and daemon
- GitHub CLI (`gh`)
- tmux
- Common build tools (git, make, curl, jq, etc.)
- `dummy` (test stub for CI testing)

The image is built from the base Ubuntu 24.04 Incus remote image and a setup script embedded
in the `clincus` binary.

---

## Building the Default Image

```bash
clincus build
```

This is a one-time setup step. The image is stored in Incus under the alias `clincus`.

### Force Rebuild

```bash
clincus build --force
```

Use this after a Clincus upgrade to pick up image updates.

---

## Listing Available Images

```bash
clincus images
```

Output:

```
NAME        ALIAS          DESCRIPTION                         CREATED
abc123def   clincus        clincus image (Docker + ...)       2026-01-01
```

The `clincus image` subcommand provides the same functionality with additional subcommands:

```bash
clincus image list
clincus image info clincus
```

---

## Custom Images

Build a custom image on top of the `clincus` base image using a shell script:

```bash
clincus build custom my-rust-image --script build-rust.sh
```

### Example Build Script

```bash
#!/bin/bash
# build-rust.sh — install Rust toolchain
set -euo pipefail

apt-get install -y build-essential
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
rustup toolchain install stable
rustup component add rustfmt clippy
```

The script runs as root inside a temporary container based on the `clincus` image (or another
`--base` you specify). When the script completes, the container is committed as a new Incus
image with the alias you provided.

### Use a Different Base Image

```bash
# Use an image from the Incus remote image server
clincus build custom my-image --base images:ubuntu/24.04 --script setup.sh

# Use another custom image as the base
clincus build custom my-extended-image --base my-rust-image --script add-more-tools.sh
```

### Force Rebuild a Custom Image

```bash
clincus build custom my-rust-image --script build-rust.sh --force
```

---

## Using a Custom Image for a Session

```bash
# One-off override
clincus shell --image my-rust-image ~/rust-project

# Set as default for a project
# .clincus.toml
# [defaults]
# image = "my-rust-image"
```

---

## Image Aliases

Incus images are referenced by fingerprint (a hash) but Clincus always works with aliases
(human-readable names). The `clincus build` command creates an alias named `clincus`
automatically. Custom images use the name you provide.

If you have multiple versions of an image, the most recently created one wins for a given
alias.

---

## Cleaning Up Images

Use the Incus CLI to remove images you no longer need:

```bash
incus image delete clincus
incus image delete my-old-image
```
