# 与 CI（Node 22）对齐；镜像源无 22 时可：`docker compose build --build-arg NODE_BUILD_IMAGE=docker.io/library/node:22-alpine frontend`
ARG NODE_BUILD_IMAGE=hub.rat.dev/library/node:24.15-alpine
ARG NGINX_RUNTIME_IMAGE=docker.io/library/nginx:alpine
FROM ${NODE_BUILD_IMAGE} AS builder

WORKDIR /app

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

COPY package.json package-lock.json ./
RUN npm ci --registry=https://registry.npmmirror.com

COPY . .
ARG VITE_MENU_MODE=simple
ENV VITE_MENU_MODE=$VITE_MENU_MODE
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
ARG VITE_STREAM_URL
ENV VITE_STREAM_URL=$VITE_STREAM_URL
RUN npm run build

FROM ${NGINX_RUNTIME_IMAGE}

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    adduser -D -u 1000 -g '' nginxuser && \
    rm -rf /etc/nginx/conf.d/default.conf

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

RUN chown -R nginxuser:nginxuser /usr/share/nginx/html /var/cache/nginx /var/log/nginx /etc/nginx

USER nginxuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8080/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
