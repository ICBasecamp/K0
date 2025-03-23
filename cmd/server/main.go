package main

import (
	"fmt"
	"os"

	"github.com/ICBasecamp/K0/internal/docker"
	"github.com/docker/docker/pkg/stdcopy"
)

func main() {

	// example usage of docker client
	dc, err := docker.CreateDockerClient()
	if err != nil {
		panic(err)
	}

	response, err := dc.StartContainer("hello-world")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Container ID: %s\n", response.ID)
	stdcopy.StdCopy(os.Stdout, os.Stderr, response.Result)

	// Make sure to close the response when done
	defer response.Result.Close()
}
