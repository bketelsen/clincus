"""
Test for clincus image publish - missing arguments.

Tests that:
1. Run clincus image publish without arguments
2. Run clincus image publish with only one argument
3. Verify usage errors
"""

import subprocess


def test_publish_no_args(clincus_binary, cleanup_containers):
    """
    Test that clincus image publish without arguments shows error.

    Flow:
    1. Run clincus image publish (no args)
    2. Verify it fails with usage message
    """
    result = subprocess.run(
        [clincus_binary, "image", "publish"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode != 0, f"Missing arguments should fail. stdout: {result.stdout}"

    combined_output = (result.stdout + result.stderr).lower()
    assert (
        "usage" in combined_output or "required" in combined_output or "argument" in combined_output
    ), f"Should show usage error. Got:\n{result.stdout + result.stderr}"


def test_publish_one_arg(clincus_binary, cleanup_containers):
    """
    Test that clincus image publish with only container shows error.

    Flow:
    1. Run clincus image publish container-name (missing alias)
    2. Verify it fails with usage message
    """
    result = subprocess.run(
        [clincus_binary, "image", "publish", "some-container"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode != 0, f"Missing alias argument should fail. stdout: {result.stdout}"

    combined_output = (result.stdout + result.stderr).lower()
    assert (
        "usage" in combined_output or "required" in combined_output or "argument" in combined_output
    ), f"Should show usage error. Got:\n{result.stdout + result.stderr}"
