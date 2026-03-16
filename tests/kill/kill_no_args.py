"""
Test for clincus kill - no container name provided.

Tests that:
1. Run clincus kill without arguments
2. Verify it fails with helpful error
"""

import subprocess


def test_kill_no_args(clincus_binary, cleanup_containers):
    """
    Test that clincus kill without arguments shows error.

    Flow:
    1. Run clincus kill (no args, no --all)
    2. Verify it fails with helpful message
    """
    result = subprocess.run(
        [clincus_binary, "kill"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode != 0, f"Kill without args should fail. stdout: {result.stdout}"

    combined_output = (result.stdout + result.stderr).lower()
    assert (
        "no container" in combined_output
        or "clincus list" in combined_output
        or "usage" in combined_output
    ), f"Should show helpful error. Got:\n{result.stdout + result.stderr}"
