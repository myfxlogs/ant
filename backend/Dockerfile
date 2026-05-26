# M6-1: multi-stage build — compile in builder, run in scratch/distroless
ARG ALPINE_MIRROR=
FROM golang:1.26.2-alpine AS builder
ARG ALPINE_MIRROR

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN if [ -n "$ALPINE_MIRROR" ]; then \
        sed -i "s|dl-cdn.alpinelinux.org|$ALPINE_MIRROR|g" /etc/apk/repositories; \
    fi && \
    apk update && \
    apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./

# go.mod uses local replaces; copy their module files before downloading deps.
COPY mt4/go.mod mt4/go.mod
COPY mt5/go.mod mt5/go.mod
RUN go mod download

COPY . .

RUN go build -ldflags="-w -s" -o /antrader ./cmd/server

FROM golang:1.26.2-alpine
ARG ALPINE_MIRROR

WORKDIR /app

RUN if [ -n "$ALPINE_MIRROR" ]; then \
        sed -i "s|dl-cdn.alpinelinux.org|$ALPINE_MIRROR|g" /etc/apk/repositories; \
    fi && \
    apk update && \
    apk add --no-cache ca-certificates tzdata wget postgresql-client procps && \
    adduser -D -u 1000 -g '' appuser

COPY --from=builder /antrader /app/antrader
COPY configs /app/configs
COPY migrations /app/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

RUN mkdir -p /var/lib/ant/spill && chmod +x /app/antrader /app/docker-entrypoint.sh && chown -R appuser:appuser /app /var/lib/ant

USER appuser

ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD sh -c "pgrep -f antrader >/dev/null 2>&1 || exit 1"

CMD ["/app/docker-entrypoint.sh"]
