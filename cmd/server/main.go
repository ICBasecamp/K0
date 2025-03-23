package main

import (
	"github.com/ICBasecamp/K0/internal/docker"
)

func main() {

	// example usage of docker client
	dc, err := docker.CreateDockerClient()
	if err != nil {
		panic(err)
	}

	response, err := dc.BuildAndStartContainer("test-image", "../test-script")
	if err != nil {
		panic(err)
	}

	docker.PrintTerminalResponse(response)
}
