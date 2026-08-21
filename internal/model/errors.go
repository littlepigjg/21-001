package model

import "errors"

var (
	ErrNotFound = errors.New("资源不存在")

	ErrAlreadyExists = errors.New("资源已存在")

	ErrInvalidArgument = errors.New("参数不合法")

	ErrUnsupportedFormat = errors.New("不支持的文件格式")

	ErrTooLarge = errors.New("文件超过大小限制")

	ErrEmptyQuery = errors.New("检索关键词不能为空")

	ErrStorage = errors.New("存储错误")
)
