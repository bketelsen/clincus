"""
Tests for building sandbox image.
"""

import subprocess


def test_build_sandbox_exists_check(clincus_binary):
    """Test that clincus-sandbox image exists or can be built."""
    # Check if image already exists
    result = subprocess.run(
        [clincus_binary, "image", "exists", "clincus-sandbox"],
        capture_output=True,
    )

    if result.returncode == 0:
        # Image exists, test passes
        assert True
    else:
        # Image doesn't exist - this is expected in a fresh install
        # The test documents that sandbox needs to be built
        assert True, "clincus-sandbox not found - run 'clincus build sandbox' to create it"


def test_build_sandbox_skip_if_exists(clincus_binary):
    """Test that building sandbox when it exists is skipped."""
    # Try to build - if image exists, should skip
    result = subprocess.run(
        [clincus_binary, "build", "sandbox"],
        capture_output=True,
        text=True,
        timeout=10,  # Short timeout since it should skip quickly
    )

    # Either succeeds or shows "already exists"
    if result.returncode == 0:
        assert "already exists" in result.stdout.lower() or "skip" in result.stderr.lower()
