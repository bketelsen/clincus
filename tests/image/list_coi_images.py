"""
Test for clincus image list - list clincus images (default behavior).

Tests that:
1. Run clincus image list
2. Verify it shows clincus images section
3. Verify output format is correct
"""

import subprocess


def test_list_coi_images(clincus_binary, cleanup_containers):
    """
    Test listing clincus images (default behavior).

    Flow:
    1. Run clincus image list
    2. Verify output contains Clincus Images section
    3. Verify clincus image is shown (exists or not built)
    """
    # === Phase 1: Run image list ===

    result = subprocess.run(
        [clincus_binary, "image", "list"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode == 0, f"Image list should succeed. stderr: {result.stderr}"

    # === Phase 2: Verify output format ===

    combined_output = result.stdout + result.stderr
    assert "Clincus Images:" in combined_output or "Available Images:" in combined_output, (
        f"Should show Clincus Images section. Got:\n{combined_output}"
    )

    # Should mention the clincus image (either built or not)
    assert "clincus" in combined_output.lower(), (
        f"Should mention clincus image. Got:\n{combined_output}"
    )
