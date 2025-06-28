# 多阶段构建 Dockerfile for FireMail
# 阶段1: 构建后端Go应用
FROM golang:1.24-alpine AS backend-builder

# 安装必要的构建工具
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app/backend

# 复制Go模块文件
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制后端源代码
COPY backend/ ./

# 构建后端应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o firemail cmd/firemail/main.go

# 验证构建结果
RUN ls -la firemail

# 阶段2: 构建前端Next.js应用
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# 安装pnpm
RUN npm install -g pnpm

# 复制package文件
COPY frontend/package.json frontend/pnpm-lock.yaml ./

# 安装依赖
RUN pnpm install --frozen-lockfile

# 复制前端源代码
COPY frontend/ ./

# 设置构建时环境变量
ENV NEXT_PUBLIC_API_BASE_URL=/api/v1
ENV NODE_ENV=production

# 构建前端应用（使用standalone模式）
RUN pnpm build

# 阶段3: 最终运行镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    sqlite-dev \
    musl-dev \
    libc6-compat \
    caddy \
    supervisor \
    tzdata \
    nodejs \
    npm

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建应用目录
WORKDIR /app

# 创建必要的目录
RUN mkdir -p /app/backend /app/frontend /app/data /app/logs /etc/supervisor/conf.d

# 从构建阶段复制文件
COPY --from=backend-builder /app/backend/firemail /app/backend/
COPY --from=backend-builder /app/backend/web /app/backend/web/
COPY --from=backend-builder /app/backend/database /app/backend/database/

# 复制前端文件 - standalone模式
COPY --from=frontend-builder /app/frontend/.next/standalone /app/frontend/
COPY --from=frontend-builder /app/frontend/.next/static /app/frontend/.next/static/
COPY --from=frontend-builder /app/frontend/public /app/frontend/public/

# 复制package.json以便Node.js能正确运行
COPY --from=frontend-builder /app/frontend/package.json /app/frontend/

# 创建Caddyfile
RUN cat > /etc/caddy/Caddyfile << 'EOF'
{
    admin off
    auto_https off
}

:3000 {
    # API代理到后端
    handle /api/* {
        reverse_proxy localhost:8080
    }

    # 直接让Next.js处理所有静态文件
    # 这样可以避免Caddy的路径匹配问题
    handle {
        reverse_proxy localhost:3001 {
            header_up Host {host}
            header_up X-Real-IP {remote}
            header_up X-Forwarded-For {remote}
            header_up X-Forwarded-Proto {scheme}
        }
    }

    # 日志
    log {
        output file /app/logs/caddy.log
        level INFO
    }
}
EOF

# 创建supervisor配置
RUN cat > /etc/supervisor/conf.d/supervisord.conf << 'EOF'
[supervisord]
nodaemon=true
user=root
logfile=/app/logs/supervisord.log
pidfile=/var/run/supervisord.pid
loglevel=info

[unix_http_server]
file=/run/supervisord.sock
chmod=0700

[supervisorctl]
serverurl=unix:///run/supervisord.sock

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[program:backend]
command=/app/backend/firemail
directory=/app/backend
autostart=true
autorestart=true
startretries=3
stderr_logfile=/app/logs/backend.log
stdout_logfile=/app/logs/backend.log
stderr_logfile_maxbytes=10MB
stdout_logfile_maxbytes=10MB
environment=HOST="0.0.0.0",PORT="8080",ENV="production",GIN_MODE="release",DB_PATH="/app/data/firemail.db",DB_BACKUP_DIR="/app/data/backups",CORS_ORIGINS="http://localhost:3000",NODE_ENV="production",NEXT_PUBLIC_API_BASE_URL="/api/v1"
redirect_stderr=true

[program:frontend]
command=node server.js
directory=/app/frontend
autostart=true
autorestart=true
startretries=3
stderr_logfile=/app/logs/frontend.log
stdout_logfile=/app/logs/frontend.log
stderr_logfile_maxbytes=10MB
stdout_logfile_maxbytes=10MB
environment=PORT="3001",HOSTNAME="0.0.0.0",NODE_ENV="production",NEXT_PUBLIC_API_BASE_URL="/api/v1"
redirect_stderr=true

[program:caddy]
command=caddy run --config /etc/caddy/Caddyfile
autostart=true
autorestart=true
startretries=3
stderr_logfile=/app/logs/caddy.log
stdout_logfile=/app/logs/caddy.log
stderr_logfile_maxbytes=10MB
stdout_logfile_maxbytes=10MB
redirect_stderr=true
EOF

# 创建启动脚本
RUN cat > /app/start.sh << 'EOF'
#!/bin/sh

echo "Starting FireMail application..."

# 创建必要的目录
mkdir -p /app/data /app/logs /app/data/backups

# 设置默认环境变量
export HOST=${HOST:-0.0.0.0}
export PORT=${PORT:-8080}
export ENV=${ENV:-production}
export GIN_MODE=${GIN_MODE:-release}
export DB_PATH=${DB_PATH:-/app/data/firemail.db}
export DB_BACKUP_DIR=${DB_BACKUP_DIR:-/app/data/backups}
export CORS_ORIGINS=${CORS_ORIGINS:-http://localhost:3000}
export NODE_ENV=${NODE_ENV:-production}
export NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL:-/api/v1}

# 设置权限
chmod 755 /app/backend/firemail
chmod -R 777 /app/data
chmod -R 755 /app/logs

# 启动supervisor
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
EOF

# 设置权限
RUN chmod +x /app/backend/firemail /app/start.sh && \
    mkdir -p /app/data /app/logs /app/data/backups && \
    chmod -R 777 /app/data && \
    chmod -R 755 /app/logs

# 暴露端口
EXPOSE 3000

# 创建数据目录的卷
VOLUME ["/app/data", "/app/logs"]

# 启动脚本
CMD ["/app/start.sh"]
