# 企业知识库全文检索系统（benzhi）

一个纯 Go 标准库实现的轻量级企业知识库全文检索后端，支持文档上传解析、分词与倒排
索引、关键词检索与相关度排序、浏览/下载统计、标签与分类管理等功能，并附带一个可
直接访问的前端页面。

- 纯 Go 标准库，无任何第三方依赖（仅 `net/http`、`sync`、`encoding/json` 等）
- 端口 `8080`，提供丰富的 RESTful API
- 支持 TXT / Markdown / PDF（简化文本抽取）
- 倒排索引 JSON 持久化，BM25 相关度排序
- 优雅关闭、健康检查、结构化日志、参数校验与统一错误处理

## 目录结构

```
.
├── cmd/server/main.go          # 服务入口（含优雅关闭）
├── internal/
│   ├── config/                 # 配置读取与校验
│   ├── handler/                # HTTP handler 与中间件
│   ├── model/                  # 数据模型
│   ├── service/                # 业务逻辑
│   └── store/                  # 存储层（内存 + JSON 持久化）
├── pkg/
│   ├── logger/                 # 结构化日志
│   ├── response/               # 统一响应格式
│   ├── tokenizer/              # 中英文分词、停用词、词干化
│   └── util/                   # 文件/时间/JSON/ID/CSV/PDF 等工具
├── web/index.html              # 前端页面（不计入 Go 代码规模）
├── go.mod
├── benzhi.Dockerfile           # 评测专用 Dockerfile
├── build_benzhi_docker.sh      # 构建脚本
└── BENZHI_README.md
```

## 本地运行

```bash
# 直接运行（默认端口 8080）
go run ./cmd/server

# 指定配置文件
go run ./cmd/server -config ./config.json
```

服务启动后访问：

- 前端页面：<http://localhost:8080/>
- 健康检查：<http://localhost:8080/health>
- 就绪检查：<http://localhost:8080/ready>

## API 文档

统一响应格式：

```json
{ "code": 0, "message": "ok", "data": {} }
```

业务码：`0` 成功、`40000` 参数错误、`40400` 不存在、`40900` 冲突、`41300` 文件过大、`50000` 内部错误。

### 健康与概览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 进程存活检查，返回版本号 |
| GET | `/ready` | 就绪检查 |
| GET | `/api/metrics` | 系统概览统计 |

### 文档管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/documents` | 分页列表（`page`、`page_size`） |
| POST | `/api/documents` | 以 JSON 创建文档 |
| POST | `/api/documents/upload` | 上传文件（multipart：`file`、`title`、`category`、`tags`） |
| GET | `/api/documents/{id}` | 文档详情（累计一次浏览） |
| PUT | `/api/documents/{id}` | 更新标题/分类/标签 |
| DELETE | `/api/documents/{id}` | 删除文档 |
| GET | `/api/documents/{id}/download` | 下载文档正文（累计一次下载） |
| GET | `/api/documents/{id}/preview` | 带高亮的摘要片段（`q`） |
| GET | `/api/documents/{id}/related` | 相关文档推荐 |

创建文档请求示例：

```json
{ "title": "标题", "content": "正文内容", "category": "技术", "tags": ["go", "检索"] }
```

### 检索

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/search` | 全文检索（`q`、`category`、`tags`、`page`、`page_size`、`sort_by`） |
| GET | `/api/search/advanced` | 高级查询（支持 `"短语"`、`category:值`、`tag:值`） |
| GET | `/api/suggest` | 检索联想（`prefix`、`limit`） |

`sort_by` 取值：`relevance`（相关度，默认）、`hot`（热度）、`time`（时间）。

检索示例：

```
GET /api/search?q=全文检索&sort_by=relevance&page=1&page_size=10
```

### 标签与分类

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tags` | 标签列表 |
| POST | `/api/tags` | 创建标签（`{"name":"..."}`） |
| DELETE | `/api/tags/{id}` | 删除标签 |
| GET | `/api/categories` | 分类列表 |
| POST | `/api/categories` | 创建分类（`{"name":"...","description":"..."}`） |
| DELETE | `/api/categories/{id}` | 删除分类 |

### 统计与导入导出

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/stats/hot` | 热度排行（`limit`） |
| GET | `/api/stats/{id}` | 单文档统计 |
| GET | `/api/export/json` | 导出全部文档为 JSON |
| GET | `/api/export/csv` | 导出全部文档为 CSV |
| POST | `/api/import` | 从 JSON 数组导入文档 |

## Docker 构建与运行

```bash
# 构建（默认镜像名 benzhi:latest，平台 linux/amd64）
./build_benzhi_docker.sh

# 自定义参数
./build_benzhi_docker.sh my-image v1.0 linux/amd64

# 运行
docker run --rm -p 8080:8080 benzhi:latest

# 进入容器执行 Go 命令（容器内保留完整 Go 工具链）
docker run --rm -it benzhi:latest bash
```

## 测试命令示例

```bash
# 编译验证
go build ./...

# 静态检查
go vet ./...

# 启动后冒烟测试
go run ./cmd/server &
curl -s http://localhost:8080/health
curl -s "http://localhost:8080/api/search?q=test"
```
