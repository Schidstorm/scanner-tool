package config

var debug string = "false"

func IsDebug() bool {
	return debug == "true"
}
