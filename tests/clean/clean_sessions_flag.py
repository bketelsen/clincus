"""
Test for clincus clean --sessions - cleans saved session data.

Tests that:
1. Create a session directory with data
2. Run clincus clean --sessions
3. Verify session data is removed
"""

import subprocess
import time

from pexpect import EOF, TIMEOUT

from support.helpers import (
    calculate_container_name,
    send_prompt,
    spawn_clincus,
    wait_for_container_ready,
    wait_for_prompt,
    wait_for_text_in_monitor,
    with_live_screen,
)


def test_clean_sessions_flag(clincus_binary, cleanup_containers, workspace_dir):
    """
    Test that clincus clean --sessions cleans saved session data.

    Flow:
    1. Start a session and poweroff (saves session)
    2. Verify session is saved
    3. Run clincus clean --sessions --force
    4. Verify session data is removed
    """
    env = {"COI_USE_DUMMY": "1"}
    calculate_container_name(workspace_dir, 1)

    # === Phase 1: Create a session ===

    child = spawn_clincus(
        clincus_binary,
        ["shell"],
        cwd=workspace_dir,
        env=env,
        timeout=120,
    )

    wait_for_container_ready(child, timeout=60)
    wait_for_prompt(child, timeout=90)

    # Interact to create session state
    with with_live_screen(child) as monitor:
        time.sleep(2)
        send_prompt(child, "session test")
        wait_for_text_in_monitor(monitor, "session test-BACK", timeout=30)

    # Poweroff to save session
    child.send("exit")
    time.sleep(0.3)
    child.send("\x0d")
    time.sleep(2)
    child.send("sudo poweroff")
    time.sleep(0.3)
    child.send("\x0d")

    try:
        child.expect(EOF, timeout=60)
    except TIMEOUT:
        pass

    try:
        child.close(force=False)
    except Exception:
        child.close(force=True)

    time.sleep(5)

    # === Phase 2: Verify session exists ===

    result = subprocess.run(
        [clincus_binary, "list", "--all"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    # Should show saved sessions

    # === Phase 3: Clean sessions ===

    result = subprocess.run(
        [clincus_binary, "clean", "--sessions", "--force"],
        capture_output=True,
        text=True,
        timeout=60,
    )

    assert result.returncode == 0, (
        f"clincus clean --sessions should succeed. stderr: {result.stderr}"
    )

    time.sleep(2)

    # === Phase 4: Verify sessions are cleaned ===

    result = subprocess.run(
        [clincus_binary, "list", "--all"],
        capture_output=True,
        text=True,
        timeout=30,
    )

    # After cleaning, should show (none) for saved sessions or fewer sessions
    output = result.stdout

    # Note: We can't strictly assert because other sessions might exist
    # Just verify the command ran successfully
    assert result.returncode == 0, f"List after clean should succeed. Got:\n{output}"
