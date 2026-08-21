# 企业知识库全文检索系统 —— 评测专用 Dockerfile
#
# 注意：本文件为评测专用，不使用多阶段构建，容器内保留完整 Go 工具链，
# 确保可以在容器内执行 go build / go run。
FROM golang:1.22

WORKDIR /app

# 复制源代码（含 go.mod、web 前端与所有 .go 文件）。
COPY . /app

# 本项目为纯 Go 实现，无 cgo 依赖。禁用 CGO 可避免在 arm64 交叉构建时
# 触发 gcc/QEMU 的 cc1 段错误，同时保证 amd64 与 arm64 均能静态编译通过。
ENV CGO_ENABLED=0

# 预先下载依赖并编译，确保镜像内依赖就绪且代码可编译。
RUN go mod download \
    && go build ./... \
    && go vet ./...

# 暴露 HTTP 服务端口。
EXPOSE 8080

# 启动服务。
CMD ["go", "run", "./cmd/server"]
