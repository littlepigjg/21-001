package model

import "errors"

// 定义业务层可复用的哨兵错误，便于上层通过 errors.Is 进行判断。
var (
	// ErrNotFound 表示目标资源不存在。
	ErrNotFound = errors.New("资源不存在")
	// ErrAlreadyExists 表示资源已存在（如重复的标签名）。
	ErrAlreadyExists = errors.New("资源已存在")
	// ErrInvalidArgument 表示参数校验失败。
	ErrInvalidArgument = errors.New("参数不合法")
	// ErrUnsupportedFormat 表示不支持的上传格式。
	ErrUnsupportedFormat = errors.New("不支持的文件格式")
	// ErrTooLarge 表示上传文件超过大小限制。
	ErrTooLarge = errors.New("文件超过大小限制")
	// ErrEmptyQuery 表示检索关键词为空。
	ErrEmptyQuery = errors.New("检索关键词不能为空")
	// ErrStorage 表示存储层发生内部错误。
	ErrStorage = errors.New("存储错误")
)
