package restart

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/start"
	"github.com/faradey/madock/v3/src/controller/general/stop"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"restart"},
		Handler:  Execute,
		Help:     "Restart containers",
		Category: "general",
	})
}

func Execute() {
	stop.Execute()
	start.Execute()
}
