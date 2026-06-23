# Strategy Worker OS Isolation (T3.3)

## Architecture

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

## RestrictedPython Status

RestrictedPython is now a **lint-only** pass.  It runs if available but does
NOT block strategy execution.  The real security boundary is at OS level.

- `validate_strategy_code()` — AST whitelist (still useful for early feedback)
- `scan_security()` — merged from sandbox_scan.py (banned modules/builtins)
- `_RestrictedEnv` — still compiles if RestrictedPython is installed, but
  strategy execution proceeds regardless of compile success

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
