# FireMail Docker Makefile

# 变量定义
IMAGE_NAME = luofengyuan/firemailplus
VERSION = latest
CONTAINER_NAME = firemail-app
COMPOSE_FILE = docker-compose.yml

# 颜色定义
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
NC = \033[0m # No Color

.PHONY: help build deploy start stop restart logs clean backup restore health

# 默认目标
help:
	@echo "$(GREEN)FireMail Docker 管理命令:$(NC)"
	@echo ""
	@echo "$(YELLOW)构建和部署:$(NC)"
	@echo "  build     - 构建Docker镜像"
	@echo "  deploy    - 部署应用 (构建+启动)"
	@echo "  start     - 启动服务"
	@echo "  stop      - 停止服务"
	@echo "  restart   - 重启服务"
	@echo ""
	@echo "$(YELLOW)管理和监控:$(NC)"
	@echo "  logs      - 查看日志"
	@echo "  health    - 检查健康状态"
	@echo "  status    - 查看服务状态"
	@echo ""
	@echo "$(YELLOW)数据管理:$(NC)"
	@echo "  backup    - 备份数据"
	@echo "  restore   - 恢复数据 (需要指定 BACKUP_DIR)"
	@echo ""
	@echo "$(YELLOW)清理:$(NC)"
	@echo "  clean     - 清理容器和镜像"
	@echo "  clean-all - 深度清理 (包括卷)"
	@echo ""
	@echo "$(YELLOW)示例:$(NC)"
	@echo "  make deploy              # 完整部署"
	@echo "  make logs                # 查看日志"
	@echo "  make backup              # 备份数据"
	@echo "  make restore BACKUP_DIR=./backup-20240101  # 恢复数据"

# 构建镜像
build:
	@echo "$(GREEN)构建Docker镜像...$(NC)"
	@chmod +x scripts/*.sh
	@./scripts/docker-build.sh

# 部署应用
deploy: build
	@echo "$(GREEN)部署FireMail应用...$(NC)"
	@./scripts/docker-deploy.sh

# 启动服务
start:
	@echo "$(GREEN)启动服务...$(NC)"
	@docker-compose up -d

# 停止服务
stop:
	@echo "$(YELLOW)停止服务...$(NC)"
	@docker-compose down

# 重启服务
restart:
	@echo "$(YELLOW)重启服务...$(NC)"
	@docker-compose restart

# 查看日志
logs:
	@echo "$(GREEN)查看日志 (Ctrl+C 退出):$(NC)"
	@docker-compose logs -f

# 查看服务状态
status:
	@echo "$(GREEN)服务状态:$(NC)"
	@docker-compose ps

# 健康检查
health:
	@echo "$(GREEN)检查服务健康状态...$(NC)"
	@curl -s http://localhost:3000/api/v1/health || echo "$(RED)健康检查失败$(NC)"
	@echo ""
	@docker-compose ps

# 备份数据
backup:
	@echo "$(GREEN)备份数据...$(NC)"
	@mkdir -p backups
	@docker cp $(CONTAINER_NAME):/app/data ./backups/backup-$(shell date +%Y%m%d-%H%M%S)
	@echo "$(GREEN)备份完成: ./backups/backup-$(shell date +%Y%m%d-%H%M%S)$(NC)"

# 恢复数据
restore:
	@if [ -z "$(BACKUP_DIR)" ]; then \
		echo "$(RED)错误: 请指定备份目录 BACKUP_DIR$(NC)"; \
		echo "$(YELLOW)示例: make restore BACKUP_DIR=./backups/backup-20240101-120000$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)恢复数据从: $(BACKUP_DIR)$(NC)"
	@docker cp $(BACKUP_DIR) $(CONTAINER_NAME):/app/data
	@docker-compose restart
	@echo "$(GREEN)数据恢复完成$(NC)"

# 清理容器和镜像
clean:
	@echo "$(YELLOW)清理容器和镜像...$(NC)"
	@docker-compose down
	@docker container prune -f
	@docker image prune -f
	@echo "$(GREEN)清理完成$(NC)"

# 深度清理
clean-all: clean
	@echo "$(RED)深度清理 (包括数据卷)...$(NC)"
	@docker-compose down -v
	@docker volume prune -f
	@docker system prune -f
	@echo "$(GREEN)深度清理完成$(NC)"

# 进入容器
shell:
	@echo "$(GREEN)进入容器...$(NC)"
	@docker-compose exec firemail sh

# 查看容器资源使用
stats:
	@echo "$(GREEN)容器资源使用情况:$(NC)"
	@docker stats $(CONTAINER_NAME) --no-stream

# 更新镜像
update:
	@echo "$(GREEN)更新镜像...$(NC)"
	@docker-compose pull
	@docker-compose up -d
	@echo "$(GREEN)更新完成$(NC)"

# 配置检查
config:
	@echo "$(GREEN)检查配置文件...$(NC)"
	@docker-compose config

# 开发模式 (挂载本地代码)
dev:
	@echo "$(GREEN)启动开发模式...$(NC)"
	@docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 生产模式
prod: deploy

# 快速重建
rebuild: stop build start

# 默认目标
.DEFAULT_GOAL := help
