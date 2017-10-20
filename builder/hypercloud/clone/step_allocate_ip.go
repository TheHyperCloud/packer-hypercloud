package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// This step checks that the network exists, and allocates an IP
// address to be attached to the instance
type stepAllocateIP struct{}

func (s *stepAllocateIP) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)

	ipName := "Packer: " + config.VMName
	ui.Say("Allocating IP address")
	ip, err := api.AllocateIP(client, config.NetworkID, ipName)
	if err != nil {
		err := fmt.Errorf("Error allocating IP via api: %s: %s", err, ip)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	address := ip["address"].(string)
	ui.Say(fmt.Sprintf("Allocated ip %s", address))
	state.Put("ip", ip)
	state.Put("ssh_address", address)

	return multistep.ActionContinue
}

func (s *stepAllocateIP) Cleanup(state multistep.StateBag) {
	// TODO: delete ip
}
