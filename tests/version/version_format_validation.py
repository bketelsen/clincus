"""
Test for clincus version - format validation.

Tests that:
1. Run clincus version
2. Verify version string matches expected format (semver)
3. Verify repository URL format
"""

import re
import subprocess


def test_version_format_validation(clincus_binary):
    """
    Test version output format with regex validation.

    Flow:
    1. Run clincus version
    2. Verify first line matches version format: clincus VERSION (commit: HASH, built: DATE)
    3. Verify second line is GitHub repository URL
    """
    result = subprocess.run(
        [clincus_binary, "version"],
        capture_output=True,
        text=True,
        timeout=10,
    )

    assert result.returncode == 0, f"Version command should succeed. stderr: {result.stderr}"

    lines = [line for line in result.stdout.strip().split("\n") if line]

    assert len(lines) == 2, f"Should have exactly 2 lines. Got:\n{result.stdout}"

    # Verify first line format: clincus VERSION (commit: HASH, built: DATE)
    # Allow various version formats:
    # - vX.Y.Z (tagged release)
    # - vX.Y.Z-N-gHASH (commits after tag)
    # - vX.Y.Z-dirty (uncommitted changes)
    # - short commit hash (development build without tags)
    version_pattern = (
        r"^clincus (v\d+\.\d+\.\d+(-\d+-g[0-9a-f]+)?(-dirty)?|[0-9a-f]+) \(commit: .+, built: .+\)$"
    )
    assert re.match(version_pattern, lines[0]), (
        f"First line should match pattern '{version_pattern}'. Got: {lines[0]}"
    )

    # Verify second line is GitHub URL
    url_pattern = r"^https://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"
    assert re.match(url_pattern, lines[1]), f"Second line should be GitHub URL. Got: {lines[1]}"
