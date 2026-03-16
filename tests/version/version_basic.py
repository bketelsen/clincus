"""
Test for clincus version - basic functionality.

Tests that:
1. Run clincus version
2. Verify version string format
3. Verify repository URL is present
"""

import subprocess


def test_version_basic(clincus_binary):
    """
    Test basic version command output.

    Flow:
    1. Run clincus version
    2. Verify exit code is 0
    3. Verify output contains version string
    4. Verify output contains repository URL
    """
    result = subprocess.run(
        [clincus_binary, "version"],
        capture_output=True,
        text=True,
        timeout=10,
    )

    assert result.returncode == 0, f"Version command should succeed. stderr: {result.stderr}"

    output = result.stdout

    # Should contain version identifier
    assert "code-on-incus (clincus) v" in output, f"Should contain version identifier. Got:\n{output}"

    # Should contain repository URL
    assert "https://github.com/mensfeld/code-on-incus" in output, (
        f"Should contain repository URL. Got:\n{output}"
    )

    # Should be exactly 2 lines
    lines = [line for line in output.strip().split("\n") if line]
    assert len(lines) == 2, f"Should output exactly 2 lines. Got {len(lines)} lines:\n{output}"
