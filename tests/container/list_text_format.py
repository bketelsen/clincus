"""
Test for clincus container list - text format output.

Tests that:
1. Container list returns text format by default
2. Output contains expected structure
"""

import subprocess


def test_container_list_text_format(clincus_binary):
    """
    Test container list with text format (default).

    Flow:
    1. Run clincus container list (without format flag)
    2. Verify text output contains expected headers
    3. Verify exit code is 0
    """
    # === Test: List containers in text format ===

    result = subprocess.run(
        [clincus_binary, "container", "list"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode == 0, (
        f"Container list should succeed. Exit code: {result.returncode}, stderr: {result.stderr}"
    )

    # Verify text output contains expected headers
    # The exact format depends on whether containers exist, but should have table structure
    output = result.stdout
    assert "NAME" in output or "No containers found" in output.lower(), (
        f"Output should contain NAME header or no containers message. Got:\n{output}"
    )
