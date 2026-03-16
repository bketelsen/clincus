# Prerequisites

Before installing Clincus, verify that your system meets the requirements below.

---

## System Requirements

| Requirement | Minimum | Notes |
|-------------|---------|-------|
| Operating system | Linux | Ubuntu 22.04+, Debian 12+, Fedora 38+, Arch, Alpine |
| Kernel | 5.15+ | Required for UID-shifting (idmapped mounts) |
| RAM | 4 GiB | 8 GiB recommended for running multiple sessions |
| Disk | 10 GiB free | The container image is ~3–4 GiB compressed |
| Architecture | amd64 or arm64 | |

!!! note "macOS"
    Clincus runs on macOS via [Colima](https://github.com/abiosoft/colima) or
    [Lima](https://lima-vm.io/). See the [macOS section](#macos-via-colima-lima) below.

---

## Install Incus

Clincus requires [Incus](https://linuxcontainers.org/incus/), the community fork of LXD.

### Ubuntu / Debian

```bash
# Add the Zabbly repository (recommended)
curl -fsSL https://pkgs.zabbly.com/key.asc | sudo gpg --dearmor -o /etc/apt/keyrings/zabbly.gpg
echo "deb [signed-by=/etc/apt/keyrings/zabbly.gpg] https://pkgs.zabbly.com/incus/stable $(lsb_release -sc) main" \
  | sudo tee /etc/apt/sources.list.d/zabbly-incus-stable.list
sudo apt update && sudo apt install incus
```

### Fedora / RHEL

```bash
sudo dnf install incus
```

### Arch Linux

```bash
sudo pacman -S incus
```

### Alpine Linux

```bash
sudo apk add incus
```

---

## Initialize Incus

After installation, run the interactive setup:

```bash
sudo incus admin init
```

For a quick non-interactive setup (suitable for development):

```bash
sudo incus admin init --minimal
```

Add your user to the `incus-admin` group so you can manage containers without `sudo`:

```bash
sudo usermod -aG incus-admin $USER
newgrp incus-admin   # apply without logging out
```

!!! warning "Group membership"
    The group change takes effect in new shell sessions. Run `newgrp incus-admin` or
    log out and back in before proceeding.

---

## Verify Incus Works

```bash
incus list
```

Expected output (empty list, no errors):

```
+------+-------+------+------+------+-----------+
| NAME | STATE | IPV4 | IPV6 | TYPE | SNAPSHOTS |
+------+-------+------+------+------+-----------+
```

If you see a permission error, ensure you are in the `incus-admin` group (see above).

---

## macOS via Colima / Lima

Clincus can run on macOS by using a Linux VM that has Incus installed.

### With Colima

```bash
# Install Colima
brew install colima

# Start a VM (uses QEMU under the hood)
colima start --vm-type qemu

# SSH into the VM and install Incus
colima ssh
# (inside VM) follow Ubuntu/Debian steps above
```

### With Lima

```bash
# Install Lima
brew install lima

# Create a VM using an Ubuntu template
limactl create --name=incus template://ubuntu
limactl start incus

# Shell into the VM
limactl shell incus
# (inside VM) follow Ubuntu/Debian steps above
```

!!! note "UID shifting on macOS VMs"
    Colima and Lima use virtiofs or 9p for file sharing, which may not support
    idmapped mounts. Set `disable_shift = true` in `~/.config/clincus/config.toml`
    if you see mount errors:

    ```toml
    [incus]
    disable_shift = true
    ```

---

## Next Steps

- [Install Clincus](install.md)
- [Quick Start](quickstart.md)
