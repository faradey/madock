package helper

import (
	"os"
)

func GetUserServiceWorkdir(service, user, workdir string) (string, string, string) {
	if os.Getenv("MADOCK_SERVICE_NAME") != "" {
		service = os.Getenv("MADOCK_SERVICE_NAME")
	}

	if os.Getenv("MADOCK_USER") != "" {
		user = os.Getenv("MADOCK_USER")
	}

	if os.Getenv("MADOCK_WORKDIR") != "" {
		workdir = os.Getenv("MADOCK_WORKDIR")
	}

	return service, user, workdir
}
