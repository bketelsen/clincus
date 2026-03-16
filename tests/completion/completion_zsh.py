"""
Test for clincus completion zsh - zsh completion generation.

Tests that:
1. Run clincus completion zsh
2. Verify it generates valid zsh completion script
3. Verify exit code is 0
"""

import subprocess


def test_completion_zsh(clincus_binary):
    """
    Test zsh completion script generation.

    Flow:
    1. Run clincus completion zsh
    2. Verify exit code is 0
    3. Verify output contains zsh completion directives
    """
    result = subprocess.run(
        [clincus_binary, "completion", "zsh"],
        capture_output=True,
        text=True,
        timeout=10,
    )

    assert result.returncode == 0, f"Completion zsh should succeed. stderr: {result.stderr}"

    output = result.stdout

    # Should contain zsh completion directives
    assert "#compdef" in output or "zsh completion" in output.lower(), (
        f"Should contain zsh completion code. Got:\n{output[:200]}..."
    )

    # Should be a substantial script
    lines = [line for line in output.split("\n") if line.strip()]
    assert len(lines) > 10, f"Should generate substantial completion script. Got {len(lines)} lines"

    # Should mention the binary name
    assert "clincus" in output.lower(), f"Should mention clincus binary. Got:\n{output[:200]}..."
