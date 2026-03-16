"""
Test for coi health --help.

Tests that:
1. Help flag shows usage information
2. Documents --format flag
3. Documents --verbose flag
4. Documents exit codes
"""

import subprocess


def test_health_help(coi_binary):
    """
    Test health command help output.

    Flow:
    1. Run coi health --help
    2. Verify usage information is shown
    3. Verify flags are documented
    """
    result = subprocess.run(
        [coi_binary, "health", "--help"],
        capture_output=True,
        text=True,
        timeout=10,
    )

    assert result.returncode == 0, f"Help should succeed. stderr: {result.stderr}"

    output = result.stdout

    # Verify usage info
    assert "health" in output, "Should show 'health' command"
    assert "Check" in output or "check" in output, "Should describe checking"

    # Verify flags are documented
    assert "--format" in output, "Should document --format flag"
    assert "--verbose" in output or "-v" in output, "Should document --verbose flag"
    assert "json" in output, "Should mention json format"

    # Verify exit codes are documented
    assert "0" in output, "Should document exit code 0"
    assert "1" in output, "Should document exit code 1"
    assert "2" in output, "Should document exit code 2"
