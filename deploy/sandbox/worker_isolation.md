# Strategy Worker OS Isolation (T3.3)

> **⚠️ 本文档已过时。** Python Worker Pool 已按 ADR-0021 退役。
> Go SDK 策略执行无需 OS 级沙箱隔离（编译为原生二进制，无外部代码解释执行）。
> 本文档留存仅供历史参考。

## Architecture（历史）

```
Host (Go scheduler)
    │
    ├─► Worker Pool (Python processes)
    │       │
    │       ├─ seccomp-bpf (syscall filter)
    │       ├─ non-root user (uid 65534)
    │       ├─ cgroup v2 (cpu.max, memory.max, pids.max)
    │       └─ net namespace (isolated per worker)
    │
    └─► gVisor/Firecracker (strong isolation for hostile multi-tenant)
```

## Isolation Layers

| Layer | Mechanism | Blocks |
|-------|-----------|--------|
| L1: Syscall | seccomp-bpf | socket, connect, fork, execve, open(write), setuid |
| L2: User | non-root (nobody:65534) | filesystem write, privileged syscalls |
| L3: Resource | cgroup v2 | CPU hog, memory leak, fork bomb |
| L4: Network | net namespace | all external network access |
| L5: VM (strong) | gVisor/Firecracker | full kernel isolation |

## Deployment Configurations

### Standard (L1–L4): Docker with seccomp + cgroup

```yaml
# docker-compose snippet
strategy-worker:
  image: anttrader/strategy-worker:latest
  user: "65534:65534"
  read_only: true
  security_opt:
    - seccomp=deploy/sandbox/seccomp_profile.json
    - no-new-privileges:true
  cap_drop:
    - ALL
  cap_add:
    - SETUID  # needed for setuid(65534)
  network_mode: "none"
  cgroup_parent: "strategy-worker.slice"
  tmpfs:
    - /tmp:noexec,nosuid,size=64M
```

### Strong (L5): gVisor (runsc)

```yaml
# Kubernetes pod spec snippet
apiVersion: v1
kind: Pod
spec:
  runtimeClassName: runsc  # gVisor
  containers:
  - name: strategy-worker
    image: anttrader/strategy-worker:latest
    securityContext:
      runAsUser: 65534
      readOnlyRootFilesystem: true
      allowPrivilegeEscalation: false
    resources:
      limits:
        cpu: "0.5"
        memory: "512Mi"
```

## Security Model

RestrictedPython has been **removed entirely**.  Strategy code runs on full
Python runtime via standard `compile()`.  All security is at OS level.

- `validate_strategy_code()` — SDK class structure + lifecycle hook checks
- `scan_security()` — AST-based banned imports/builtins scan (lint only)
- `apply_os_sandbox()` — cgroup v2 + seccomp-bpf + drop_root (kernel enforces)
- `_apply_resource_limits()` — RLIMIT_AS/CPU/NOFILE (kernel enforces)

## Verification

```bash
# Check seccomp status
grep Seccomp /proc/self/status

# Check cgroup
cat /proc/self/cgroup

# Run escape tests (informational in dev, blocking in CI with seccomp)
python3 -m pytest strategy-service/tests/test_sandbox_escape.py -v

# Deploy with gVisor
kubectl apply -f deploy/sandbox/gvisor-pod.yaml
```
