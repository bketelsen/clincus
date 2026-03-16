"""
Test for clincus run - executes with UID 1000.

Tests that:
1. Run id command
2. Verify UID is 1000
"""

import subprocess


def test_run_uid_1000(clincus_binary, cleanup_containers, workspace_dir):
    """
    Test that commands run with UID 1000.

    Flow:
    1. Run clincus run id
    2. Verify UID is 1000
    """
    result = subprocess.run(
        [clincus_binary, "run", "--workspace", workspace_dir, "--", "id", "-u"],
        capture_output=True,
        text=True,
        timeout=180,
    )

    assert result.returncode == 0, f"Run should succeed. stderr: {result.stderr}"

    combined_output = result.stdout + result.stderr
    assert "1000" in combined_output, f"Should run with UID 1000. Got:\n{combined_output}"
