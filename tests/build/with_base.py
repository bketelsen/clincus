"""
Integration tests for custom image building.

Tests:
- clincus build custom with script
- Custom image with base specified
- Custom image with privileged base
"""

import json
import subprocess


def test_build_custom_with_base(clincus_binary, tmp_path):
    """Test building a custom image with explicit base."""
    image_name = "clincus-test-custom-base"

    # Create build script
    build_script = tmp_path / "build_base.sh"
    build_script.write_text("""#!/bin/bash
set -e
apt-get update
apt-get install -y jq
""")

    # Build custom image with ubuntu:22.04 as base
    result = subprocess.run(
        [
            clincus_binary,
            "build",
            "custom",
            image_name,
            "--script",
            str(build_script),
            "--base",
            "images:ubuntu/24.04",
        ],
        capture_output=True,
        text=True,
        timeout=300,
    )
    assert result.returncode == 0, f"Build with base failed: {result.stderr}"

    # Verify JSON output
    output = json.loads(result.stdout)
    assert output["alias"] == image_name

    # Cleanup
    subprocess.run([clincus_binary, "image", "delete", image_name], check=False)
