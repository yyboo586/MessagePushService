#!/bin/bash

set -e

# 配置变量
SERVICE_NAME="message_push_service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
INSTALL_DIR="/usr/share/${SERVICE_NAME}"
BINARY_NAME="message_push_service"
CONFIG_FILE="config.yaml"

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] $1"
}

error() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [ERROR] $1" >&2
}

warn() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [WARN] $1" >&2
}

# 检查是否为root用户
check_root() {
  if [ "$EUID" -ne 0 ]; then
    error "此脚本需要root权限运行, 请使用 sudo 或切换到root用户"
    exit 1
  fi
}

# 检查必要文件是否存在
check_files() {
  log "[1] 检查必要文件"
  
  if [ ! -f "./bin/${BINARY_NAME}" ]; then
    error "可执行文件不存在: ./bin/${BINARY_NAME}"
    error "请先运行 build.sh 编译项目"
    exit 1
  fi
  
  if [ ! -f "./${CONFIG_FILE}" ]; then
    error "配置文件不存在: ./${CONFIG_FILE}"
    exit 1
  fi
  
  if [ ! -f "./${SERVICE_NAME}.service" ]; then
    error "systemd服务文件不存在: ./${SERVICE_NAME}.service"
    exit 1
  fi
  
  log "所有必要文件检查通过"
}

# 检查服务是否已安装
is_service_installed() {
  systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service" || return 1
}

# 检查服务是否正在运行
is_service_running() {
  systemctl is-active --quiet "${SERVICE_NAME}" || return 1
}

# 安装systemd服务
install_service() {
  log "[2] 安装systemd服务"
  
  if is_service_installed; then
    log "服务已安装，更新服务文件"
  else
    log "首次安装服务"
  fi
  
  # 复制服务文件
  cp "./${SERVICE_NAME}.service" "${SERVICE_FILE}"
  
  # 重新加载systemd配置
  systemctl daemon-reload
  
  # 启用服务（开机自启）
  systemctl enable "${SERVICE_NAME}"
  
  log "服务安装/更新完成"
}

# 停止服务（如果正在运行）
stop_service() {
  if is_service_running; then
    log "[3] 停止服务"
    systemctl stop "${SERVICE_NAME}"
    
    # 等待服务完全停止
    local count=0
    while is_service_running && [ $count -lt 10 ]; do
      sleep 1
      count=$((count + 1))
    done
    
    if is_service_running; then
      error "服务停止超时，强制终止"
      systemctl kill "${SERVICE_NAME}" || true
    else
      log "服务已成功停止"
    fi
  else
    log "[3] 服务未运行，跳过停止步骤"
  fi
}

# 创建安装目录
create_install_dir() {
  log "[4] 创建安装目录"
  if [ ! -d "${INSTALL_DIR}" ]; then
    mkdir -p "${INSTALL_DIR}"
    log "创建目录: ${INSTALL_DIR}"
  else
    log "目录已存在: ${INSTALL_DIR}"
  fi
}

# 部署文件
deploy_files() {
  log "[5] 部署文件"
  
  # 备份现有文件（如果存在）
  if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
    cp "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}.backup.$(date +%Y%m%d_%H%M%S)" || true
  fi
  
  # 复制新文件
  cp "./bin/${BINARY_NAME}" "${INSTALL_DIR}/"
  cp "./${CONFIG_FILE}" "${INSTALL_DIR}/"
  
  # 设置可执行权限
  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
  
  # 设置文件所有者
  chown -R root:root "${INSTALL_DIR}"
  
  log "文件部署完成"
}

# 启动服务
start_service() {
  log "[6] 启动服务"
  
  if is_service_installed; then
    systemctl start "${SERVICE_NAME}"
    
    # 等待服务启动
    sleep 2
    
    if is_service_running; then
      log "服务启动成功"
    else
      error "服务启动失败"
      systemctl status "${SERVICE_NAME}" --no-pager
      exit 1
    fi
  else
    error "服务未安装，无法启动"
    exit 1
  fi
}

# 检查服务状态
check_service_status() {
  log "[7] 检查服务状态"
  systemctl status "${SERVICE_NAME}" --no-pager
  
  if is_service_running; then
    log "✅ 服务运行正常"
  else
    error "❌ 服务运行异常"
    exit 1
  fi
}

# 显示服务信息
show_service_info() {
  log "[8] 服务信息"
  echo "服务名称: ${SERVICE_NAME}"
  echo "安装目录: ${INSTALL_DIR}"
  echo "服务文件: ${SERVICE_FILE}"
  echo ""
  echo "常用命令:"
  echo "  查看状态: systemctl status ${SERVICE_NAME}"
  echo "  查看日志: journalctl -u ${SERVICE_NAME} -f"
  echo "  停止服务: systemctl stop ${SERVICE_NAME}"
  echo "  启动服务: systemctl start ${SERVICE_NAME}"
  echo "  重启服务: systemctl restart ${SERVICE_NAME}"
}

# 主函数
main() {
  log "开始部署 ${SERVICE_NAME}"
  
  check_root
  check_files
  install_service
  stop_service
  create_install_dir
  deploy_files
  start_service
  check_service_status
  show_service_info
  
  log "🎉 部署完成！"
}

# 执行主函数
main "$@"