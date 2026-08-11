# Runbook · MdGatewayClockSkewHigh

> 关联告警：`MdGatewayClockSkewHigh`（`md_clock_skew_max_seconds > 30`，持续 2m，severity=warning）

## 症状

`md_clock_skew_max_seconds > 30` 持续触发。服务器时钟与 broker/MT 服务器时钟偏差超过 30 秒。

## 影响

bar 时间戳不准确 → bar 归属到错误的时间段 → 回测/实盘 K线数据时间轴偏移。策略依赖时间判断（如"仅在特定时段交易"）可能行为异常。

## 诊断步骤

```bash
# 1. 查看当前时钟同步状态
timedatectl status

# 2. 查 NTP 同步详情
chronyc tracking 2>/dev/null || ntpq -p 2>/dev/null

# 3. 确认系统时间与 UTC 偏差
date -u && docker exec alphaforge-backend date -u

# 4. 查后端日志中的时钟偏差记录
docker logs alphaforge-backend --since 10m | grep -iE "clock.*skew|time.*drift|ntp"
```

## 应急处置

1. **NTP 服务未运行** → 启动 NTP 同步：
   ```bash
   sudo systemctl start chronyd || sudo systemctl start ntpd
   sudo timedatectl set-ntp true
   ```
2. **NTP 同步但偏差大** → 强制立即同步：
   ```bash
   sudo chronyc makestep  # chrony
   # 或
   sudo ntpd -gq          # ntp
   ```
3. **容器时区不对** → 确认 Docker 容器使用 UTC：
   ```bash
   docker exec alphaforge-backend cat /etc/timezone
   # 应为 Etc/UTC
   ```
4. **NTP 服务器不可达** → 检查防火墙是否阻止 NTP（UDP 123）出站，更换 NTP 服务器

## 常见根因

- **NTP 服务停止**：`chronyd` 或 `ntpd` 服务崩溃/未启动，系统时钟漂移
- **NTP 服务器不可达**：防火墙阻止 UDP 123 出站，或 NTP 服务器地址配置错误
- **容器时区配置错误**：Docker 容器未挂载宿主机时区，使用默认 UTC 但宿主机时区不对
- **宿主机休眠/恢复**：VM 休眠后时钟漂移，NTP 未及时追赶
