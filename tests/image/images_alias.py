"""
Test for clincus images - alias for clincus image list.

Tests that:
1. Run clincus images
2. Verify it behaves like clincus image list
"""

import subprocess


def test_images_alias(clincus_binary, cleanup_containers):
    """
    Test that 'clincus images' is an alias for 'clincus image list'.

    Flow:
    1. Run clincus images
    2. Verify output is similar to clincus image list
    """
    # === Phase 1: Run clincus images ===

    result = subprocess.run(
        [clincus_binary, "images"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode == 0, f"clincus images should succeed. stderr: {result.stderr}"

    # === Phase 2: Verify output format ===

    combined_output = result.stdout + result.stderr

    # Should show same content as image list
    assert "Clincus Images:" in combined_output or "Available Images:" in combined_output, (
        f"Should show Clincus Images section. Got:\n{combined_output}"
    )


def test_images_all_flag(clincus_binary, cleanup_containers):
    """
    Test that 'clincus images --all' works.

    Flow:
    1. Run clincus images --all
    2. Verify it shows all local images
    """
    # === Phase 1: Run clincus images --all ===

    result = subprocess.run(
        [clincus_binary, "images", "--all"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode == 0, f"clincus images --all should succeed. stderr: {result.stderr}"

    # === Phase 2: Verify output ===

    combined_output = result.stdout + result.stderr

    # Should show All Local Images section
    assert "All Local Images:" in combined_output or "ALIAS" in combined_output, (
        f"Should show All Local Images section. Got:\n{combined_output}"
    )
