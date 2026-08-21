#!/usr/bin/env bash
#
# 构建 benzhi 评测镜像。
#
# 用法：
#   ./build_benzhi_docker.sh [镜像名] [标签] [平台]
#
# 默认参数：
#   镜像名 = benzhi
#   标签   = latest
#   平台   = linux/amd64
set -euo pipefail

IMAGE_NAME="${1:-benzhi}"
IMAGE_TAG="${2:-latest}"
PLATFORM="${3:-linux/amd64}"

FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

echo "==> 检查 Docker 是否可用..."
if ! command -v docker >/dev/null 2>&1; then
  echo "错误：未检测到 docker，请先安装 Docker。" >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "错误：Docker 守护进程未运行或当前用户无权限访问。" >&2
  exit 1
fi

echo "==> 构建镜像：${FULL_IMAGE}（平台：${PLATFORM}）"
docker build \
  --platform "${PLATFORM}" \
  -f benzhi.Dockerfile \
  -t "${FULL_IMAGE}" \
  .

echo ""
echo "构建成功：${FULL_IMAGE}"
echo ""
echo "运行示例："
echo "  docker run --rm -p 8080:8080 ${FULL_IMAGE}"
echo ""
echo "进入容器执行 go 命令："
echo "  docker run --rm -it ${FULL_IMAGE} bash"
echo ""
echo "启动后访问："
echo "  健康检查 http://localhost:8080/health"
echo "  前端页面 http://localhost:8080/"
