package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// This step advanced creates the instance, and attaches all resources
type stepBuildInstance struct{}

func (s *stepBuildInstance) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)
	boot_disk := state.Get("disk").(map[string]interface{})
	ip := state.Get("ip").(map[string]interface{})

	instanceName := "Packer: " + config.VMName

	diskids := []string{
		boot_disk["id"].(string),
	}
	ipids := []string{
		ip["id"].(string),
	}

	ui.Say("Creating instance...")
	instance, err := api.InstanceCreate(client, instanceName, config.Memory,
		config.InstancePerformanceTierID, config.regionId, diskids, ipids, "disk")

	if err != nil {
		err := fmt.Errorf("Error creating instance: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	ui.Say(fmt.Sprintf("Instance created with ID: %s", instance["id"]))
	state.Put("instance", instance)
	return multistep.ActionContinue
}

func (s *stepBuildInstance) Cleanup(state multistep.StateBag) {
	// TODO: terminate instance
}
