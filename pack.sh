#!/bin/bash

set -e

# 日志函数
log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] $1"
}

error() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [ERROR] $1" >&2
}

# 定义产物目录
DIST_DIR="./dist"
PACKAGE_NAME="message_push_service_$(date '+%Y%m%d_%H%M%S')"
PACKAGE_DIR="${DIST_DIR}/${PACKAGE_NAME}"

log "[1] 清理旧的编译文件和产物目录..."
rm -rf ./bin/message_push_service
rm -rf ${DIST_DIR}

log "[2] 编译可执行文件..."
if ! go build -o ./bin/message_push_service main.go; then
  error "编译失败，请检查代码"
  exit 1
fi

log "[3] 检查编译结果..."
if [ ! -f "./bin/message_push_service" ]; then
  error "可执行文件不存在，编译可能失败"
  exit 1
fi

log "[4] 创建产物目录..."
mkdir -p ${PACKAGE_DIR}

log "[5] 拷贝文件到产物目录..."
cp ./bin/message_push_service ${PACKAGE_DIR}/
cp ./config.yaml ${PACKAGE_DIR}/ || { error "config.yaml 不存在"; exit 1; }
cp ./deploy.sh ${PACKAGE_DIR}/
cp ./message_push_service.service ${PACKAGE_DIR}/

log "[6] 压缩打包..."
cd ${DIST_DIR}
tar -czf ${PACKAGE_NAME}.tar.gz ${PACKAGE_NAME}
cd ..

log "[7] 清理临时目录..."
rm -rf ${PACKAGE_DIR}

log "[8] 打包完成！"
log "[9] 产物位置: ${DIST_DIR}/${PACKAGE_NAME}.tar.gz"