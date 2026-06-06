package cmd

import "os"

func coopConfigFolder() string {
	return Config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
}
