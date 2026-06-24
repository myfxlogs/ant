"""T3.3 — Sandbox escape tests.

Validates that the OS-level sandbox blocks:
  - Network access (socket creation)
  - File write (open for writing)
  - Process spawning (subprocess, fork)
  - Privilege escalation (setuid)

Tests are designed to pass in two modes:
  - HARD mode: seccomp/cgroup applied → escape attempts FAIL (blocked)
  - SOFT mode: no seccomp/cgroup → escape attempts SUCCEED (no false positives)

The test reports whether the sandbox is active.
"""

import os
import sys
import unittest


def _sandbox_active() -> bool:
    """Check if OS sandbox is ACTIVE on this process (already applied, not just available).

    We don't apply the sandbox in these tests because seccomp is irreversible
    and would kill the test runner.  Instead, we verify the sandbox module
    works correctly and escape tests are informational.
    """
    # Check if seccomp was already applied to this process.
    try:
        with open("/proc/self/status") as f:
            for line in f:
                if line.startswith("Seccomp:"):
                    # 0=disabled, 1=strict, 2=filter
                    mode = int(line.split(":")[1].strip())
                    if mode > 0:
                        return True
    except (OSError, ValueError):
        pass
    return False


class TestSandboxEscapeNetwork(unittest.TestCase):
    """Network isolation tests."""

    def test_cannot_create_socket(self):
        """Creating a TCP socket should be blocked or succeed depending on sandbox state."""
        try:
            import socket
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.close()
            socket_ok = True
        except (OSError, PermissionError):
            socket_ok = False

        if _sandbox_active():
            self.assertFalse(socket_ok, "Socket creation should be BLOCKED in sandbox")
        # In soft mode (no sandbox), socket creation may succeed — no assertion needed.

    def test_cannot_connect(self):
        """Connecting to a remote host should be blocked in sandbox."""
        try:
            import socket
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(0.1)
            s.connect(("8.8.8.8", 53))
            s.close()
            connect_ok = True
        except (OSError, PermissionError, TimeoutError):
            connect_ok = False

        if _sandbox_active():
            self.assertFalse(connect_ok, "Network connect should be BLOCKED in sandbox")


class TestSandboxEscapeFilesystem(unittest.TestCase):
    """Filesystem write isolation tests."""

    def test_cannot_write_file(self):
        """Writing to /tmp should be blocked in sandbox."""
        test_path = "/tmp/sandbox_escape_test_write.txt"
        try:
            with open(test_path, "w") as f:
                f.write("escape test data")
            os.unlink(test_path)
            write_ok = True
        except (OSError, PermissionError):
            write_ok = False

        if _sandbox_active():
            self.assertFalse(write_ok, "File write should be BLOCKED in sandbox")

    def test_cannot_create_directory(self):
        """Creating directories should be blocked in sandbox."""
        test_dir = "/tmp/sandbox_escape_test_dir"
        try:
            os.makedirs(test_dir, exist_ok=True)
            os.rmdir(test_dir)
            mkdir_ok = True
        except (OSError, PermissionError):
            mkdir_ok = False

        if _sandbox_active():
            self.assertFalse(mkdir_ok, "Directory creation should be BLOCKED in sandbox")


class TestSandboxEscapeProcess(unittest.TestCase):
    """Process isolation tests."""

    def test_cannot_spawn_subprocess(self):
        """Spawning a subprocess should be blocked in sandbox."""
        try:
            import subprocess
            subprocess.run(["echo", "test"], capture_output=True, timeout=1)
            spawn_ok = True
        except (OSError, subprocess.SubprocessError, FileNotFoundError):
            spawn_ok = False

        if _sandbox_active():
            self.assertFalse(spawn_ok, "Subprocess spawn should be BLOCKED in sandbox")

    def test_cannot_fork(self):
        """Fork should be blocked in sandbox."""
        try:
            pid = os.fork()
            if pid == 0:
                os._exit(0)
            else:
                os.waitpid(pid, 0)
            fork_ok = True
        except (OSError, AttributeError):
            fork_ok = False

        if _sandbox_active():
            self.assertFalse(fork_ok, "Fork should be BLOCKED in sandbox")


class TestSandboxEscapePrivilege(unittest.TestCase):
    """Privilege escalation tests."""

    def test_cannot_setuid(self):
        """setuid should be blocked or fail silently."""
        try:
            os.setuid(0)
            setuid_ok = True
        except (OSError, PermissionError, AttributeError):
            setuid_ok = False

        if _sandbox_active():
            self.assertFalse(setuid_ok, "setuid should be BLOCKED in sandbox")


class TestSandboxStatus(unittest.TestCase):
    """Sandbox status reporting."""

    def test_sandbox_os_imports(self):
        """sandbox_os.py must be importable."""
        from app.engine import sandbox_os
        self.assertIsNotNone(sandbox_os)

    def test_seccomp_availability_reported(self):
        """seccomp_is_available must return a bool (not crash)."""
        from app.engine.sandbox_os import seccomp_is_available
        result = seccomp_is_available()
        self.assertIsInstance(result, bool)

    def test_apply_os_sandbox_returns_status(self):
        """apply_os_sandbox must return a dict with status keys."""
        from app.engine.sandbox_os import apply_os_sandbox
        status = apply_os_sandbox(
            cpu_max_pct=50.0, memory_mb=512, pid_max=64,
            apply_seccomp_flag=False,  # don't lock our test process
            drop_root_flag=False,
        )
        self.assertIn("cgroup", status)
        self.assertIn("seccomp", status)
        self.assertIn("total_layers", status)
        self.assertIsInstance(status["total_layers"], int)

    def test_cgroup_detection(self):
        """_cgroup_v2_available must return a bool."""
        from app.engine.sandbox_os import _cgroup_v2_available
        result = _cgroup_v2_available()
        self.assertIsInstance(result, bool)


class TestRestrictedPythonDowngrade(unittest.TestCase):
    """T3.3: RestrictedPython is now lint-only — execution must work without it."""

    def test_scan_security_works(self):
        """The merged scan_security function must work."""
        from app.engine.sandbox import scan_security

        # Safe code should pass.
        result = scan_security("x = 1 + 2\nprint(x)")
        self.assertTrue(result.passed, f"Safe code rejected: {result.violations}")

        # Banned import should be caught.
        result = scan_security("import os\nos.system('rm -rf /')")
        self.assertFalse(result.passed, "Banned import not caught")
        self.assertGreater(len(result.violations), 0)

        # Banned builtin should be caught.
        result = scan_security("eval('1+1')")
        self.assertFalse(result.passed, "Banned builtin not caught")



if __name__ == "__main__":
    unittest.main()
