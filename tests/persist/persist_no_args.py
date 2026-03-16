"""
Test for clincus persist - error handling when no arguments provided.

Tests that:
1. Run clincus persist with no arguments
2. Verify error message about no containers specified
3. Verify helpful message pointing to 'clincus list'
"""

import subprocess


def test_persist_no_args(clincus_binary):
    """
    Test persist command with no arguments.

    Flow:
    1. Run persist with no container names
    2. Verify error about no containers specified
    3. Verify helpful message
    """

    # === Phase 1: Run persist with no arguments ===

    result = subprocess.run(
        [clincus_binary, "persist"],
        capture_output=True,
        text=True,
        timeout=60,
    )

    # Should fail with non-zero exit code
    assert result.returncode != 0, "Should fail when no containers specified"

    combined_output = result.stdout + result.stderr

    # Should mention no containers provided
    assert (
        "no container" in combined_output.lower()
        or "no names" in combined_output.lower()
        or "required" in combined_output.lower()
    ), f"Should show error about no containers. Got:\n{combined_output}"

    # Should point to 'clincus list' for help
    assert "clincus list" in combined_output, (
        f"Should suggest 'clincus list' to see active containers. Got:\n{combined_output}"
    )
