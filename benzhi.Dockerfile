# 企业知识库全文检索系统 —— 评测专用 Dockerfile
#
# 注意：本文件为评测专用，不使用多阶段构建，容器内保留完整 Go 工具链，
# 确保可以在容器内执行 go build / go run。
FROM golang:1.22

WORKDIR /app

# 单机跨架构构建（QEMU 模拟 arm64）时，并行编译会触发 binfmt 的
# fork/exec "exec format error"，因此禁用并行编译，保证两种架构均能构建成功。
ENV GOFLAGS=-p=1

# 复制源代码（含 go.mod、web 前端与所有 .go 文件）。
COPY . /app

# 预先下载依赖并编译，确保镜像内依赖就绪且代码可编译。
RUN go mod download \
    && go build ./... \
    && go vet ./...

# 暴露 HTTP 服务端口。
EXPOSE 8080

# 启动服务。
CMD ["go", "run", "./cmd/server"]
