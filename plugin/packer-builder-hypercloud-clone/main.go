package main

import (
	"github.com/mitchellh/packer/packer/plugin"
	"github.com/thehypercloud/packer-hypercloud/builder/hypercloud/clone"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(clone.Builder))
	server.Serve()
}
