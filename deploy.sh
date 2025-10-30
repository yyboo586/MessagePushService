#!/bin/bash

set -e

# 日志函数
log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] $1"
}

error() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [ERROR] $1" >&2
}

# 配置
INSTALL_DIR="/usr/share/message_push_service"
SERVICE_NAME="message_push_service"

# 检查是否以root权限运行
if [ "$EUID" -ne 0 ]; then 
  error "请使用root权限运行此脚本"
  exit 1
fi

log "================================================"
log "开始部署 ${SERVICE_NAME} 服务..."
log "================================================"

# 检查是否为首次部署
if [ ! -d "${INSTALL_DIR}" ]; then
  log "================================================"
  log "检测到首次部署，创建安装目录..."
  log "================================================"
  mkdir -p ${INSTALL_DIR}
  FIRST_DEPLOY=true
else
  log "================================================"
  log "检测到已有安装，执行升级部署..."
  log "================================================"
  FIRST_DEPLOY=false
fi

# 如果服务已存在，先停止
if systemctl list-units --full -all | grep -Fq "${SERVICE_NAME}.service"; then
  systemctl stop ${SERVICE_NAME} || true
fi

# 备份旧文件（升级场景）
if [ "${FIRST_DEPLOY}" = false ] && [ -f "${INSTALL_DIR}/message_push_service" ]; then
  BACKUP_DIR="${INSTALL_DIR}/backup_$(date '+%Y%m%d_%H%M%S')"
  log "================================================"
  log "备份当前版本到 ${BACKUP_DIR}..."
  log "================================================"
  mkdir -p ${BACKUP_DIR}
  cp -r ${INSTALL_DIR}/* ${BACKUP_DIR}/ 2>/dev/null || true
fi

log "================================================"
log "拷贝文件到 ${INSTALL_DIR}..."
log "================================================"
cp ./message_push_service ${INSTALL_DIR}/
chmod +x ${INSTALL_DIR}/message_push_service
cp ./config.yaml ${INSTALL_DIR}/config.yaml


# 检查systemd服务文件是否存在
if [ ! -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
  if [ -f "./message_push_service.service" ]; then
    log "================================================"
    log "发现服务配置文件，正在安装..."
    log "================================================"
    cp ./message_push_service.service /etc/systemd/system/${SERVICE_NAME}.service
  else
    error "================================================"
    error "未找到 message_push_service.service 文件, 请手动配置systemd服务"
    error "================================================"
    exit 1
  fi
fi

log "================================================"
log "重新加载 systemd 配置..."
log "================================================"
systemctl daemon-reload
systemctl enable ${SERVICE_NAME}

# 启动服务
log "================================================"
log "启动 ${SERVICE_NAME} 服务..."
log "================================================"
systemctl start ${SERVICE_NAME}

# 等待服务启动
sleep 2

# 检查服务状态
log "================================================"
log "检查服务状态..."
log "================================================"
if systemctl is-active --quiet ${SERVICE_NAME}; then
  log "✓ ${SERVICE_NAME} 服务已成功启动并运行中"
  systemctl status ${SERVICE_NAME} --no-pager
  exit 0
else
  error "✗ ${SERVICE_NAME} 服务启动失败"
  log "查看详细日志: journalctl -u ${SERVICE_NAME} -n 50"
  systemctl status ${SERVICE_NAME} --no-pager || true
  exit 1
fi
