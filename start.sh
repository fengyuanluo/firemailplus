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
