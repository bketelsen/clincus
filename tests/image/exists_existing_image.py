"""
Test for clincus image exists - check existing image.

Tests that:
1. Check for clincus image (should exist after build)
2. Verify exit code is 0
"""

import subprocess


def test_exists_coi_image(clincus_binary, cleanup_containers):
    """
    Test checking if the clincus image exists.

    Flow:
    1. Run clincus image exists clincus
    2. Verify exit code is 0 (image exists)

    Note: This test assumes the clincus image has been built.
    """
    # === Phase 1: Check if clincus image exists ===

    result = subprocess.run(
        [clincus_binary, "image", "exists", "clincus"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    # === Phase 2: Verify success ===

    assert result.returncode == 0, f"clincus image should exist. stderr: {result.stderr}"
