"""
Test main CLI -h shorthand.

Expected:
- -h works as shorthand for --help
"""

import subprocess


def test_main_help_shorthand(clincus_binary):
    """Test that clincus -h works as shorthand for --help."""
    result = subprocess.run([clincus_binary, "-h"], capture_output=True, text=True, timeout=5)

    assert result.returncode == 0
    assert "code-on-incus" in result.stdout.lower()
