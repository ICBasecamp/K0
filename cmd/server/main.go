package main

import (

	// "github.com/ICBasecamp/K0/internal/docker"
	"github.com/ICBasecamp/K0/internal/s3"
)

func main() {

	// example usage of docker client
	// dc, err := docker.CreateDockerClient()
	// if err != nil {
	// 	panic(err)
	// }

	// response, err := dc.BuildAndStartContainer("test-image", "../test-script")
	// if err != nil {
	// 	panic(err)
	// }

	// docker.PrintTerminalResponse(response)

	// example usage of s3 client
	s3c, err := s3.CreateS3Client()
	if err != nil {
		panic(err)
	}

	_, err = s3c.ListObjects()
	if err != nil {
		panic(err)
	}

}
