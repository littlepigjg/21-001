package util

// Version 是系统版本号。
const Version = "1.0.0"

// BuildInfo 描述构建信息，可在编译时通过 -ldflags 注入。
var BuildInfo = "development"

// VersionString 返回可读的版本描述。
func VersionString() string {
	return "benzhi-" + Version + " (" + BuildInfo + ")"
}
