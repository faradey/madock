package restart

import (
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/stop"
)

func Execute() {
	stop.Execute()
	start.Execute()
}
