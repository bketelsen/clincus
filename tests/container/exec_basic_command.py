"""
Test for clincus container exec - basic command execution.

Tests that:
1. Launch a container
2. Execute a simple command
3. Verify output
"""

import subprocess
import time

from support.helpers import (
    calculate_container_name,
)


def test_exec_basic_command(clincus_binary, cleanup_containers, workspace_dir):
    """
    Test basic command execution in container.

    Flow:
    1. Launch a container
    2. Execute echo command
    3. Verify output contains expected text
    4. Cleanup
    """
    container_name = calculate_container_name(workspace_dir, 1)

    # === Phase 1: Launch container ===

    result = subprocess.run(
        [clincus_binary, "container", "launch", "clincus", container_name],
        capture_output=True,
        text=True,
        timeout=120,
    )

    assert result.returncode == 0, f"Container launch should succeed. stderr: {result.stderr}"

    time.sleep(3)

    # === Phase 2: Execute command ===

    result = subprocess.run(
        [clincus_binary, "container", "exec", container_name, "--", "echo", "hello-test-123"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode == 0, f"Exec should succeed. stderr: {result.stderr}"

    # === Phase 3: Verify output ===

    combined_output = result.stdout + result.stderr
    assert "hello-test-123" in combined_output, (
        f"Output should contain echo text. Got:\n{combined_output}"
    )

    # === Phase 4: Cleanup ===

    subprocess.run(
        [clincus_binary, "container", "delete", container_name, "--force"],
        capture_output=True,
        timeout=30,
    )
