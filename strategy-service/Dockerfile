FROM python:3.11-slim

WORKDIR /app

# 安装系统依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    git \
	iptables \
	gosu \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 10001 app && useradd -m -u 10001 -g 10001 -s /usr/sbin/nologin app

# 复制依赖文件
COPY requirements.txt .

# 安装Python依赖
RUN pip install --no-cache-dir -r requirements.txt

# 复制应用代码
COPY app/ app/

COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

RUN mkdir -p /app/data/memory && chown -R 10001:10001 /app

# 环境变量
ENV HOST=0.0.0.0
ENV PORT=8081
ENV DEBUG=false
ENV EXECUTION_TIMEOUT=30
ENV MAX_MEMORY_MB=256
ENV MAX_BACKTEST_WORKERS=2
ENV BACKTEST_TIMEOUT=120
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV HOME=/tmp
ENV XDG_CACHE_HOME=/tmp/.cache

# 暴露端口
EXPOSE 8081

# 启动应用
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["sh", "-lc", "python -m uvicorn app.main:app --host 0.0.0.0 --port ${PORT:-8081}"]
