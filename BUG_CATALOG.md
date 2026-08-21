# 缺陷候选清单（BUG_CATALOG）

> 项目：企业知识库全文检索系统（benzhi）
>
> 说明：本清单仅标注「可注入缺陷的位置」，当前代码为无缺陷基线版本。每个缺陷均为
> 后端（Go）运行时缺陷，注入后项目仍能编译通过并正常运行，且能稳定复现。缺陷序号
> 统一为 001~030。其中 10 个缺陷涉及 3 个后端文件，其余 20 个缺陷涉及 2 个后端文件；
> 每个缺陷的注入均需修改 50 行以上代码，且涉及并发、context 生命周期、错误传播、运行时
> 崩溃、状态污染或跨层数据流等运行时机制，避免「枚举/关键字映射」类单点数据变换错误。

| bug_id | bug_category | 缺陷描述 | 植入位置 | 修改行数 | 预期表现 | 触发方式 | 缺陷难度 |
|--------|--------------|----------|----------|----------|----------|----------|----------|
| benzhi-concur-001 | concurrency | 倒排索引 map 无锁并发读写导致数据竞争 | internal/service/index_service.go.BuildIndex、internal/store/index_store.go.AddPosting、internal/service/search_service.go.collectCandidates | >50 行 | 并发上传时 panic "concurrent map read and map write" | 同时上传多份文档触发并发建索引 | ★★★ |
| benzhi-concur-002 | concurrency | 浏览次数采用非原子读改写导致计数丢失 | internal/service/document_service.go.GetDocument、internal/store/stats_store.go.IncrementView、internal/service/stats_service.go.IncrementView | >50 行 | 浏览次数增长小于实际请求数 | 并发 GET 同一文档详情 | ★★★★ |
| benzhi-concur-003 | concurrency | 标签关联计数并发累加丢失更新 | internal/service/upload_service.go.UploadDocument、internal/store/tag_store.go.BumpTagCount、internal/service/tag_service.go.EnsureTag | >50 行 | 标签 Count 与实际关联文档数不符 | 并发上传多个同标签文档 | ★★★ |
| benzhi-concur-004 | concurrency | 排序读取统计与自增并发导致热度值不一致 | internal/service/search_service.go.buildHits、internal/store/stats_store.go.GetStats | >50 行 | 热度排序结果抖动或数据竞争 | 检索同时高频浏览文档 | ★★★★ |
| benzhi-concur-005 | concurrency | 文档删除与索引移除非原子导致残留 | internal/service/document_service.go.DeleteDocument、internal/store/index_store.go.RemoveDocumentFromIndex | >50 行 | 检索仍返回已删除文档 | 删除文档同时并发检索 | ★★★★ |
| benzhi-concur-006 | concurrency | 限流器令牌桶 map 无锁并发访问 | internal/handler/rate_limiter.go.allow、internal/handler/middleware.go.withMiddleware | >50 行 | 高频请求 panic 或限流失效 | 高频并发请求触发限流器 | ★★★ |
| benzhi-nil-007 | nil | 索引 map 未初始化时直接写入词项 | internal/store/index_store.go.AddPosting、internal/store/store.go.Load | >50 行 | panic "assignment to entry in nil map" | 加载空索引后上传文档 | ★★ |
| benzhi-nil-008 | nil | 统计记录缺失时返回 nil 指针被解引用 | internal/service/search_service.go.buildHits、internal/store/stats_store.go.GetStats、internal/service/stats_service.go.GetDocumentStats | >50 行 | panic "invalid memory address or nil pointer dereference" | 无统计记录的文档参与检索 | ★★ |
| benzhi-nil-009 | nil | 装了 nil 指针的接口与 nil 比较为假 | internal/service/tag_service.go.EnsureTag、internal/service/upload_service.go.UploadDocument、internal/store/tag_store.go.GetTagByName | >50 行 | 误判标签已存在导致关联错误 | 上传含新标签的文档 | ★★★★ |
| benzhi-nil-010 | nil | 导入数据含 nil 元素时未校验直接解引用 | internal/service/export_service.go.ImportDocumentsJSON、internal/store/document_store.go.CreateDocument | >50 行 | panic nil pointer dereference | 导入包含空元素的 JSON 数组 | ★★ |
| benzhi-nil-011 | nil | 统计 map 未初始化时向 nil map 自增 | internal/store/stats_store.go.IncrementView、internal/service/stats_service.go.IncrementView | >50 行 | panic "assignment to entry in nil map" | 新文档首次浏览时统计 map 为 nil | ★★ |
| benzhi-slice-012 | slice | append 复用底层数组污染已入索引位置 | internal/service/index_service.go.BuildIndex、internal/store/index_store.go.addPostingLocked、internal/service/search_service.go.Search | >50 行 | 词项位置错乱或词频错误 | 上传多篇共享词项的文档 | ★★★★ |
| benzhi-slice-013 | slice | 返回文档复用内部 Tags 切片导致污染 | internal/store/document_store.go.GetDocument、internal/service/document_service.go.UpdateDocument | >50 行 | 更新元数据后标签被意外篡改 | 更新文档标签后查看原文档 | ★★★ |
| benzhi-slice-014 | slice | 分页偏移未校验导致切片越界 | internal/service/search_service.go.paginate、internal/handler/search_handler.go.Search、internal/service/document_service.go.ListDocuments | >50 行 | panic "slice bounds out of range" | 请求超大页码 | ★★ |
| benzhi-slice-015 | slice | RemoveTag 复用原数组导致删除残留 | internal/model/document.go.RemoveTag、internal/service/tag_service.go.DeleteTag | >50 行 | 文档仍显示已删除标签 | 删除标签后再次查看文档 | ★★★ |
| benzhi-slice-016 | slice | 倒排列表子切片写回共享污染其他词项 | internal/store/index_store.go.RemoveDocumentFromIndex、internal/service/index_service.go.RemoveIndex | >50 行 | 删除文档后检索结果异常 | 删除文档后检索 | ★★★★ |
| benzhi-error-017 | error | %w 丢失导致 errors.Is 判断失效 | internal/store/store.go.Load、internal/service/document_service.go.CreateDocument | >50 行 | 错误分类错误返回 500 而非 400 | 触发底层存储错误 | ★★★ |
| benzhi-error-018 | error | 内层 err 被 := 遮蔽导致错误被忽略 | internal/service/upload_service.go.UploadDocument、internal/handler/upload_handler.go.UploadDocument、internal/store/document_store.go.CreateDocument | >50 行 | 上传失败但接口返回成功 | 上传过程中某一步失败 | ★★★★ |
| benzhi-error-019 | error | 用错误字符串比较替代 errors.Is | internal/handler/document_handler.go.GetDocument、internal/service/document_service.go.GetDocument | >50 行 | 错误处理分支失效返回错误状态码 | 请求不存在的文档 | ★★ |
| benzhi-error-020 | error | 删除时忽略清理错误导致不一致 | internal/service/document_service.go.DeleteDocument、internal/store/document_store.go.DeleteDocument | >50 行 | 删除成功但索引残留检索到已删文档 | 删除文档 | ★★★ |
| benzhi-error-021 | error | 索引构建错误被上传主流程忽略导致文档与索引状态不一致 | internal/service/upload_service.go.UploadDocument、internal/service/index_service.go.BuildIndex | >50 行 | 上传成功但检索不到该文档 | 上传时索引持久化失败 | ★★★ |
| benzhi-context-022 | context | 请求取消未向下游存储遍历传播 | internal/handler/document_handler.go.ListDocuments、internal/service/document_service.go.ListDocuments | >50 行 | 客户端取消后服务端仍继续处理 | 客户端超时取消列表请求 | ★★★ |
| benzhi-context-023 | context | 请求 context 存入结构体后被跨请求复用 | internal/service/service.go.New、internal/handler/handler.go.NewHandler | >50 行 | 复用已取消 context 导致请求异常 | 首次请求取消后再次请求 | ★★★★ |
| benzhi-context-024 | context | 检索循环忽略 ctx.Err() 继续评分 | internal/service/search_service.go.Search、internal/handler/search_handler.go.Search、internal/service/rank_service.go.scoreDocument | >50 行 | 取消后仍持续消耗资源 | 检索大结果集时取消 | ★★★ |
| benzhi-context-025 | context | 优雅关闭超时未处理导致进程挂起 | cmd/server/main.go.main、internal/store/store.go.Close | >50 行 | 关闭超时后进程未退出 | 优雅关闭时连接未释放 | ★★★★ |
| benzhi-defer-026 | defer | 循环内 defer 释放资源直到函数结束才执行 | internal/service/export_service.go.ImportDocumentsJSON、internal/store/document_store.go.CreateDocument、internal/service/index_service.go.BuildIndex | >50 行 | 批量导入时资源/文件句柄耗尽 | 导入大批量文档 | ★★★ |
| benzhi-defer-027 | defer | defer 错误修改命名返回值 | internal/service/reindex_service.go.RebuildIndex、internal/store/index_store.go.FlushIndex | >50 行 | 返回错误的词项计数 | 重建索引 | ★★★★ |
| benzhi-defer-028 | defer | error 分支跳过文件句柄释放 | internal/service/upload_service.go.UploadDocument、internal/handler/upload_handler.go.UploadDocument | >50 行 | 文件句柄泄漏 | 重复上传失败文件 | ★★ |
| benzhi-other-029 | other | 时间戳单位混用导致排序错误 | internal/service/search_service.go.sortHits、internal/store/document_store.go.ListDocuments | >50 行 | 按时间排序结果错误 | 按上传时间排序检索 | ★★★ |
| benzhi-other-030 | other | BM25 分母为零产生 NaN 评分 | internal/service/rank_service.go.scoreDocument、internal/service/search_service.go.buildHits | >50 行 | 评分 NaN 导致排序错乱 | 空文档集或极端参数检索 | ★★★ |
