package main

import "github.com/mitchellh/packer/packer/plugin"
import "github.com/thehypercloud/packer-hypercloud/builder/hypercloud/vnc"

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(vnc.Builder))
	server.Serve()
}
