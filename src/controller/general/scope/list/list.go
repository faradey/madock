package list

import (
	"fmt"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
)

func Execute() {
	scopes := configs.GetScopes(configs.GetProjectName())
	for key, val := range scopes {
		fmtc.Title(key)
		if val == "1" {
			fmtc.SuccessLn(" active")
		} else {
			fmt.Println("")
		}
	}
}
