"""
Meta test for full installation process.

This test acts as a smoke test for the entire installation workflow:
1. Launch a fresh Ubuntu 24.04 container
2. Install Incus inside it (nested Incus)
3. Follow the README installation steps
4. Build the clincus binary
5. Verify clincus --help works
6. Verify basic clincus commands work

This validates the complete setup process from scratch.

Note: This test requires nested Incus support and takes longer to run.
"""

import os
import subprocess
import time

import pytest


@pytest.fixture(scope="module")
def meta_container():
    """
    Launch a fresh Ubuntu container to test the installation process.

    This validates that the README installation steps work correctly
    and produce a functioning clincus binary.
    """
    container_name = "clincus-meta-test"

    # Clean up any existing test container
    subprocess.run(
        ["incus", "delete", container_name, "--force"],
        capture_output=True,
        check=False,
    )

    # Launch fresh Ubuntu 24.04 container
    result = subprocess.run(
        [
            "incus",
            "launch",
            "images:ubuntu/24.04",
            container_name,
        ],
        capture_output=True,
        text=True,
        timeout=180,
    )

    if result.returncode != 0:
        pytest.skip(f"Failed to launch meta container: {result.stderr}")

    # Wait for container to be ready
    time.sleep(10)

    yield container_name

    # Cleanup
    subprocess.run(
        ["incus", "delete", container_name, "--force"],
        capture_output=True,
        check=False,
    )


def exec_in_container(container_name, command, timeout=300, check=True):
    """Execute command in meta container and return result."""
    result = subprocess.run(
        ["incus", "exec", container_name, "--", "bash", "-c", command],
        capture_output=True,
        text=True,
        timeout=timeout,
        check=check,
    )
    return result


def test_full_installation_process(meta_container, clincus_binary):
    """
    Test the complete installation process from README.

    This is a smoke test that validates:
    1. System dependencies can be installed
    2. Go can be installed
    3. Repository can be cloned
    4. clincus binary can be built from source
    5. clincus --help works
    6. clincus version works

    This does NOT test Incus functionality - it only validates the
    build process and that the binary executes correctly.
    """
    container_name = meta_container

    # Phase 1: Install system dependencies
    # Retry apt-get operations to handle transient network issues in CI
    max_retries = 3
    last_error = None

    for attempt in range(max_retries):
        result = exec_in_container(
            container_name,
            """
            set -e
            # Wait for network and DNS to be ready
            for i in {1..30}; do
                if ping -c 1 archive.ubuntu.com >/dev/null 2>&1; then
                    break
                fi
                sleep 1
            done

            # Update package lists with retry
            apt-get update -qq || sleep 5 && apt-get update -qq

            # Install packages
            DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
                curl wget git ca-certificates gnupg build-essential libsystemd-dev

            echo "System dependencies installed"
            """,
            timeout=600,
            check=False,
        )

        if result.returncode == 0:
            break

        last_error = result.stderr
        if attempt < max_retries - 1:
            print(f"apt-get attempt {attempt + 1} failed, retrying...")
            time.sleep(10)  # Wait before retry

    assert result.returncode == 0, (
        f"Failed to install dependencies after {max_retries} attempts: {last_error}"
    )

    # Phase 2: Install Go
    result = exec_in_container(
        container_name,
        """
        set -e
        GO_VERSION="1.21.13"
        wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
        rm -rf /usr/local/go
        tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
        rm go${GO_VERSION}.linux-amd64.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
        /usr/local/go/bin/go version
        """,
        timeout=300,
    )
    assert result.returncode == 0, f"Failed to install Go: {result.stderr}"
    assert "go version" in result.stdout, "Go installation verification failed"

    # Phase 3: Clone repository and build clincus
    # In CI (pull requests), try PR branch first, fall back to default branch
    # This handles:
    # - Fork PRs (branch doesn't exist in main repo)
    # - Deleted branches (branch was deleted after PR merged)
    github_branch = os.environ.get("GITHUB_HEAD_REF", "")
    github_repo = os.environ.get("GITHUB_REPOSITORY", "bketelsen/clincus")
    github_server = os.environ.get("GITHUB_SERVER_URL", "https://github.com")
    repo_url = f"{github_server}/{github_repo}.git"

    # Build clone command with fallback: try branch first, then default
    if github_branch:
        clone_script = f"""
        if git clone -b {github_branch} {repo_url} 2>/dev/null; then
            echo "Cloned branch {github_branch}"
        else
            echo "Branch {github_branch} not found, cloning default branch"
            git clone {repo_url}
        fi
        """
    else:
        clone_script = f"git clone {repo_url}"

    result = exec_in_container(
        container_name,
        f"""
        set -e
        cd /root
        {clone_script}
        cd clincus
        /usr/local/go/bin/go build -o clincus ./cmd/clincus
        ./clincus version
        """,
        timeout=300,
    )
    assert result.returncode == 0, f"Failed to build clincus: {result.stderr}"
    assert "clincus " in result.stdout, "clincus version check failed"

    # Phase 4: Test clincus --help
    result = exec_in_container(
        container_name,
        """
        cd /root/clincus
        ./clincus --help
        """,
        timeout=30,
    )
    assert result.returncode == 0, f"clincus --help failed: {result.stderr}"
    assert "clincus is a CLI tool" in result.stdout, (
        "clincus help output missing expected text"
    )
    assert "Available Commands:" in result.stdout, "clincus help missing commands section"

    # Phase 5: Test clincus basic commands
    result = exec_in_container(
        container_name,
        """
        cd /root/clincus
        ./clincus images --help
        ./clincus list --help
        ./clincus shell --help
        echo "Basic commands work"
        """,
        timeout=30,
    )
    assert result.returncode == 0, f"Basic clincus commands failed: {result.stderr}"


def test_installation_with_prebuilt_binary(meta_container, clincus_binary):
    """
    Test installation using pre-built binary (simpler workflow).

    This tests the path where users download a pre-built binary
    instead of building from source. No Incus installation needed,
    just validates the binary executes correctly.

    Flow:
    1. Copy pre-built clincus binary into container
    2. Test clincus --help works
    3. Test clincus version works
    """
    container_name = meta_container

    # Push pre-built binary to container
    result = subprocess.run(
        ["incus", "file", "push", clincus_binary, f"{container_name}/usr/local/bin/clincus"],
        capture_output=True,
        text=True,
        timeout=30,
    )
    assert result.returncode == 0, f"Failed to push binary: {result.stderr}"

    # Make executable and test
    result = exec_in_container(
        container_name,
        """
        chmod +x /usr/local/bin/clincus
        clincus --help
        clincus version
        """,
        timeout=30,
    )
    assert result.returncode == 0, f"Pre-built binary test failed: {result.stderr}"
    assert "clincus" in result.stdout, "clincus binary not working correctly"
