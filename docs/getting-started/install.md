# Installation

Choose your preferred installation method.

---

## Package Manager (recommended)

Binary packages for Debian/Ubuntu (`.deb`), Fedora/RHEL (`.rpm`), and Alpine (`.apk`) are
available on [GitHub Releases](https://github.com/bketelsen/clincus/releases).

### Debian / Ubuntu

```bash
VERSION=v0.1.0   # replace with latest version
curl -fsSL "https://github.com/bketelsen/clincus/releases/download/${VERSION}/clincus_${VERSION}_linux_amd64.deb" \
  -o clincus.deb
sudo dpkg -i clincus.deb
```

### Fedora / RHEL

```bash
VERSION=v0.1.0
sudo rpm -i "https://github.com/bketelsen/clincus/releases/download/${VERSION}/clincus_${VERSION}_linux_amd64.rpm"
```

### Alpine

```bash
VERSION=v0.1.0
wget "https://github.com/bketelsen/clincus/releases/download/${VERSION}/clincus_${VERSION}_linux_amd64.apk"
sudo apk add --allow-untrusted clincus_${VERSION}_linux_amd64.apk
```

---

## Binary Download

Download the pre-built binary for your architecture:

```bash
VERSION=v0.1.0
ARCH=amd64   # or arm64
curl -fsSL "https://github.com/bketelsen/clincus/releases/download/${VERSION}/clincus_${VERSION}_linux_${ARCH}.tar.gz" \
  | tar -xz clincus
sudo install -m 755 clincus /usr/local/bin/clincus
```

Verify the installation:

```bash
clincus version
# clincus v0.1.0 (commit: abc1234, built: 2026-01-01T00:00:00Z)
```

---

## Build from Source

Requirements: Go 1.24+, Node.js 20+, `make`.

```bash
git clone https://github.com/bketelsen/clincus.git
cd clincus
make install
```

`make install` builds the binary (including the embedded Svelte web dashboard) and copies it
to `$GOPATH/bin/clincus`.

To build without installing:

```bash
make build
# produces ./clincus binary
```

---

## Shell Completions

Clincus can generate shell completion scripts for bash, zsh, and fish.

### Bash

```bash
clincus completion bash | sudo tee /etc/bash_completion.d/clincus > /dev/null
# or for the current user only:
clincus completion bash >> ~/.bash_completion
```

### Zsh

```bash
clincus completion zsh > "${fpath[1]}/_clincus"
# then restart your shell or run:
autoload -U compinit && compinit
```

### Fish

```bash
clincus completion fish > ~/.config/fish/completions/clincus.fish
```

You can also use `make completions` in the source repository to generate all completion
scripts at once into the `completions/` directory.

---

## Next Steps

- [Quick Start](quickstart.md) — build your first image and run a session
