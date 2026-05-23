#!/bin/sh
set -eu

# Best-effort egress deny for sensitive internal services.
# NOTE: iptables requires root even with NET_ADMIN.
iptables -P OUTPUT DROP || true
iptables -A OUTPUT -o lo -j ACCEPT || true

# Allow established/related.
iptables -A OUTPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT || true

# Allow DNS.
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT || true
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT || true

# Block backend/trading and MT gateways explicitly (defense-in-depth).
iptables -A OUTPUT -p tcp --dport 8080 -j REJECT || true
iptables -A OUTPUT -p tcp --dport 5000 -j REJECT || true
iptables -A OUTPUT -p tcp --dport 5001 -j REJECT || true

# Allow outbound to nothing else by default.

# Fix memory dir permissions after volume mount (volume owner may be root)
mkdir -p /app/data/memory && chown -R 10001:10001 /app/data/memory || true

exec gosu 10001:10001 "$@"
