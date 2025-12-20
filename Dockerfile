# 多阶段构建 Dockerfile for FireMail（精简运行时，保留功能）

# 阶段1: 构建后端Go应用
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o firemail cmd/firemail/main.go

# 阶段2: 构建前端Next.js应用（standalone）
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
RUN npm install -g pnpm
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
ENV NEXT_PUBLIC_API_BASE_URL=/api/v1
ENV NODE_ENV=production
RUN pnpm build

# 阶段3: 运行镜像（单容器运行前后端）
FROM node:20-alpine
RUN apk add --no-cache ca-certificates sqlite tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app

# 复制后端可执行文件和数据库文件
COPY --from=backend-builder /app/backend/firemail /app/backend/firemail
COPY --from=backend-builder /app/backend/database /app/backend/database
COPY --from=backend-builder /app/backend/web /app/backend/web

# 复制前端 standalone 产物
COPY --from=frontend-builder /app/frontend/.next/standalone /app/frontend
COPY --from=frontend-builder /app/frontend/.next/static /app/frontend/.next/static
COPY --from=frontend-builder /app/frontend/public /app/frontend/public
COPY --from=frontend-builder /app/frontend/package.json /app/frontend/package.json

# 创建启动脚本
RUN cat > /app/start.sh << 'EOF' && chmod +x /app/start.sh
#!/bin/sh
set -e

mkdir -p /app/data /app/logs /app/data/backups
chmod -R 777 /app/data || true
chmod -R 755 /app/logs || true
chmod +x /app/backend/firemail

# 启动后端
cd /app/backend
HOST=$HOST PORT=$BACKEND_PORT /app/backend/firemail &
cd /app

# 启动前端
cd /app/frontend
PORT=$FRONTEND_PORT HOSTNAME=0.0.0.0 NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL exec node server.js
EOF

ENV BACKEND_PORT=8080
ENV FRONTEND_PORT=3000
ENV HOST=0.0.0.0
ENV ENV=production
ENV GIN_MODE=release
ENV DB_PATH=/app/data/firemail.db
ENV DB_BACKUP_DIR=/app/data/backups
ENV CORS_ORIGINS=http://localhost:3000
ENV NODE_ENV=production
ENV NEXT_PUBLIC_API_BASE_URL=/api/v1

EXPOSE 3000 8080
VOLUME ["/app/data", "/app/logs"]
CMD ["/app/start.sh"]
