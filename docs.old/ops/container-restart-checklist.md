# 容器重建与重启检查清单

## 1. 修改后固定检查

涉及代码修改后执行：

```bash
python3 scripts/check-file-lines.py
cd backend && go test ./...
npm run build --prefix frontend
make verify
git diff --check
```

## 2. 重建与重启

```bash
docker compose up -d --build backend frontend
```

如只更新单个服务且已确认依赖无变化，可使用 `--no-deps`，但最终批次验收应重建 backend 和 frontend。

## 3. 健康检查

```bash
docker compose ps
curl -fsS http://localhost:8012/health
curl -fsS http://localhost:8012/
```

## 4. 失败处理

- 构建失败先看 Docker build 输出，不直接删除缓存或数据卷。
- 服务不健康先查 `docker compose logs --tail=200 <service>`。
- 数据库、Redis、MT 网关异常不得用重置数据规避。

## 5. 验收记录

每批整改完成后记录：

- 行数检查结果。
- 后端测试结果。
- 前端构建结果。
- `make verify` 结果。
- 容器健康状态。
