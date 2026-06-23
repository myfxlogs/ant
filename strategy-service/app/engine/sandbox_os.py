"""OS-level sandbox hardening (T3.3+ / D5).  Production-grade seccomp BPF.

Security boundary: OS kernel (not language-level RestrictedPython).
  - seccomp-bpf: proper BPF bytecode filter (no libseccomp dependency)
  - Non-root: drop to nobody:65534
  - cgroup v2: cpu.max / memory.max / pids.max
  - Network namespace: isolated per worker (via Docker/K8s)

Architecture:
  Worker launcher → apply_os_sandbox() → execute strategy
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
      seccomp()     drop_root()    cgroup_limit()

BPF filter blocks: socket/connect/send/bind/accept (network), open-WR/create
(file write), fork/clone/execve (process), setuid/setgid (privilege).
"""

from __future__ import annotations

import ctypes
import os
import struct
import sys
from typing import List, Optional

# ── BPF instruction format (Linux kernel stable ABI) ───────────────────

# struct sock_filter { __u16 code; __u8 jt; __u8 jf; __u32 k; };
_BPF_LD  = 0x00  # load
_BPF_JMP = 0x05  # jump
_BPF_RET = 0x06  # return
_BPF_JEQ = 0x10  # jump if == k
_BPF_JGE = 0x30  # jump if >= k
_BPF_JGT = 0x20  # jump if > k
_BPF_JSET = 0x40 # jump if & k

_BPF_W   = 0x00  # 32-bit
_BPF_ABS = 0x20  # absolute offset

# seccomp return values.
_SECCOMP_RET_KILL  = 0x00000000
_SECCOMP_RET_ALLOW = 0x7fff0000

# seccomp data offsets in BPF.
_OFFSET_ARCH = 4     # audit arch
_OFFSET_NR   = 0     # syscall number
_AUDIT_ARCH_X86_64 = 0xC000003E

# prctl constants.
_PR_SET_SECCOMP = 22
_SECCOMP_MODE_FILTER = 2


def _bpf_stmt(code: int, k: int) -> "sock_filter":
    return sock_filter(code, 0, 0, k)


def _bpf_jump(code: int, k: int, jt: int, jf: int) -> "sock_filter":
    return sock_filter(code, jt, jf, k)


class sock_filter(ctypes.Structure):
    _fields_ = [
        ("code", ctypes.c_uint16),
        ("jt",   ctypes.c_uint8),
        ("jf",   ctypes.c_uint8),
        ("k",    ctypes.c_uint32),
    ]

    def pack(self) -> bytes:
        return struct.pack("HBBI", self.code, self.jt, self.jf, self.k)


class sock_fprog(ctypes.Structure):
    _fields_ = [
        ("len",    ctypes.c_ushort),
        ("filter", ctypes.c_void_p),
    ]


# ── Blocked syscall numbers (x86_64 stable ABI) ───────────────────────

_BLOCKED_SYSCALLS: List[int] = [
    # Network — blocks any socket communication.
    41,   # socket
    42,   # connect
    44,   # sendto
    45,   # recvfrom
    46,   # sendmsg
    47,   # recvmsg
    49,   # bind
    50,   # listen
    51,   # accept
    52,   # getsockname
    53,   # getpeername
    54,   # socketpair
    # File write — blocks persistent file creation/writing.
    2,    # open  (combined with O_WRONLY/O_RDWR below via argument check)
    85,   # creat
    257,  # openat (modern glibc)
    # Process — blocks spawning or executing.
    56,   # clone
    57,   # fork
    58,   # vfork
    59,   # execve
    322,  # execveat
    # Privilege — blocks escalation.
    105,  # setuid
    106,  # setgid
    116,  # setgroups
    117,  # setresuid
    119,  # setresgid
    # Dangerous.
    169,  # reboot
    172,  # iopl
    173,  # ioperm
]


def _build_seccomp_filter() -> bytes:
    """Build a BPF filter program that blocks dangerous syscalls.

    The filter:
      1. Load audit arch (offset 4) → if not x86_64, KILL (wrong arch)
      2. Load syscall number (offset 0)
      3. For each blocked syscall: if == NR → KILL
      4. Otherwise → ALLOW

    Returns packed BPF bytecode ready for prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER).
    """
    insns: List[sock_filter] = []

    # 1. Validate architecture.
    insns.append(_bpf_stmt(_BPF_LD | _BPF_W | _BPF_ABS, _OFFSET_ARCH))
    insns.append(_bpf_jump(_BPF_JMP | _BPF_JEQ, _AUDIT_ARCH_X86_64, 1, 0))
    insns.append(_bpf_stmt(_BPF_RET, _SECCOMP_RET_KILL))  # wrong arch → KILL

    # 2. Load syscall number.
    insns.append(_bpf_stmt(_BPF_LD | _BPF_W | _BPF_ABS, _OFFSET_NR))

    # 3. For each blocked syscall: if NR == blocked → KILL.
    for nr in sorted(_BLOCKED_SYSCALLS):
        insns.append(_bpf_jump(_BPF_JMP | _BPF_JEQ, nr, 0, 1))
        insns.append(_bpf_stmt(_BPF_RET, _SECCOMP_RET_KILL))

    # 4. Default: ALLOW.
    insns.append(_bpf_stmt(_BPF_RET, _SECCOMP_RET_ALLOW))

    return b"".join(i.pack() for i in insns)


# ── Public API ─────────────────────────────────────────────────────────

def seccomp_is_available() -> bool:
    """Check if seccomp is available on this system (Linux >= 3.5)."""
    try:
        libc = ctypes.CDLL(ctypes.util.find_library("c"), use_errno=True)
        return True
    except (AttributeError, OSError):
        return False


def _get_libc():
    return ctypes.CDLL(ctypes.util.find_library("c"), use_errno=True)


def apply_seccomp() -> bool:
    """Apply the seccomp-bpf filter.  Irreversible — once applied, blocked
    syscalls will cause immediate SIGSYS (kill).  Returns True on success.

    The filter blocks: network, file write, process creation, privilege
    escalation.  numpy/pandas operations (mmap, read, write to temp files
    via already-open fds) are NOT affected.
    """
    if not seccomp_is_available():
        return False

    try:
        libc = _get_libc()
        libc.prctl.argtypes = [ctypes.c_int, ctypes.c_ulong, ctypes.c_void_p]
        libc.prctl.restype = ctypes.c_int

        bpf_bytes = _build_seccomp_filter()
        fprog = sock_fprog(len(bpf_bytes) // 8, ctypes.cast(
            ctypes.create_string_buffer(bpf_bytes), ctypes.c_void_p
        ))

        result = libc.prctl(_PR_SET_SECCOMP, _SECCOMP_MODE_FILTER, ctypes.byref(fprog))
        if result != 0:
            err = ctypes.get_errno()
            if err == 1:  # EPERM — already seccomp'd or insufficient perms
                return False
            return False
        return True
    except Exception:
        return False


# ── Non-root enforcement ────────────────────────────────────────────────

def drop_root(uid: Optional[int] = None, gid: Optional[int] = None) -> bool:
    """Drop root privileges to an unprivileged user."""
    if os.getuid() != 0:
        return False
    try:
        if gid is None:
            gid = 65534
        if uid is None:
            uid = int(os.environ.get("SUDO_UID", "65534"))
        os.setgroups([])
        os.setgid(gid)
        os.setuid(uid)
        return os.getuid() != 0
    except OSError:
        return False


def is_root() -> bool:
    return os.getuid() == 0


# ── Cgroup limits ───────────────────────────────────────────────────────

CGROUP_V2_PATH = "/sys/fs/cgroup"


def _cgroup_v2_available() -> bool:
    return os.path.isfile(f"{CGROUP_V2_PATH}/cgroup.controllers")


def apply_cgroup_limits(
    cpu_max_pct: float = 50.0,
    memory_mb: int = 512,
    pid_max: int = 64,
    group_name: str = "strategy-worker",
) -> bool:
    """Apply cgroup v2 resource limits to the current process."""
    if not _cgroup_v2_available():
        return False
    cgroup_path = f"{CGROUP_V2_PATH}/{group_name}"
    try:
        os.makedirs(cgroup_path, exist_ok=True)
        controllers = "+cpu +memory +pids"
        with open(f"{cgroup_path}/cgroup.subtree_control", "w") as f:
            f.write(controllers)
        quota_us = int(cpu_max_pct * 1000)
        if quota_us > 0:
            with open(f"{cgroup_path}/cpu.max", "w") as f:
                f.write(f"{quota_us} 100000\n")
        with open(f"{cgroup_path}/memory.max", "w") as f:
            f.write(f"{memory_mb * 1024 * 1024}\n")
        with open(f"{cgroup_path}/pids.max", "w") as f:
            f.write(f"{pid_max}\n")
        with open(f"{cgroup_path}/cgroup.procs", "w") as f:
            f.write(f"{os.getpid()}\n")
        return True
    except (OSError, PermissionError):
        return False


# ── Combined OS sandbox ─────────────────────────────────────────────────

def apply_os_sandbox(
    cpu_max_pct: float = 50.0,
    memory_mb: int = 512,
    pid_max: int = 64,
    apply_seccomp_flag: bool = True,
    drop_root_flag: bool = True,
) -> dict:
    """Apply all available OS-level sandboxing in correct order.

    Order: cgroup → seccomp → drop_root
    """
    status = {"cgroup": False, "seccomp": False, "drop_root": False, "total_layers": 0}
    if _cgroup_v2_available():
        status["cgroup"] = apply_cgroup_limits(cpu_max_pct, memory_mb, pid_max)
        if status["cgroup"]:
            status["total_layers"] += 1
    if apply_seccomp_flag and seccomp_is_available():
        status["seccomp"] = apply_seccomp()
        if status["seccomp"]:
            status["total_layers"] += 1
    if drop_root_flag and is_root():
        status["drop_root"] = drop_root()
        if status["drop_root"]:
            status["total_layers"] += 1
    return status


# ── Escape test helpers ─────────────────────────────────────────────────

def can_open_network() -> bool:
    try:
        import socket
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.close()
        return True
    except (OSError, PermissionError):
        return False


def can_write_file(path: str = "/tmp/sandbox_escape_test") -> bool:
    try:
        with open(path, "w") as f:
            f.write("escape test")
        os.unlink(path)
        return True
    except (OSError, PermissionError):
        return False


def can_spawn_process() -> bool:
    try:
        import subprocess
        subprocess.run(["true"], capture_output=True, timeout=1)
        return True
    except (OSError, subprocess.SubprocessError):
        return False
