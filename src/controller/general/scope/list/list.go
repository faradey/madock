package list

import (
	"fmt"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/cli/output"
	"github.com/faradey/madock/src/helper/configs"
)

type ScopeListOutput struct {
	Scopes []ScopeInfo `json:"scopes"`
	Active string      `json:"active"`
}

type ScopeInfo struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"scope:list"},
		Handler:  Execute,
		Help:     "List scopes. Supports --json (-j) output",
		Category: "scope",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralScopeList)).(*arg_struct.ControllerGeneralScopeList)

	scopes := configs.GetScopes(configs.GetProjectName())

	if args.Json {
		var scopeList []ScopeInfo
		var activeScope string
		for key, val := range scopes {
			isActive := val == "1"
			scopeList = append(scopeList, ScopeInfo{
				Name:   key,
				Active: isActive,
			})
			if isActive {
				activeScope = key
			}
		}
		output.PrintJSON(ScopeListOutput{
			Scopes: scopeList,
			Active: activeScope,
		})
		return
	}

	for key, val := range scopes {
		fmtc.Title(key)
		if val == "1" {
			fmtc.SuccessLn(" active")
		} else {
			fmt.Println("")
		}
	}
}
