package util

const Version = "1.0.0"

var BuildInfo = "development"

func VersionString() string {
	return "benzhi-" + Version + " (" + BuildInfo + ")"
}
