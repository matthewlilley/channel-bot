package build

var Commit string

const version = "1.2.0"

func GetVersion() string {
	return "v" + version + "-" + Commit
}
