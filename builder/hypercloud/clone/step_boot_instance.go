package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// Boot the instance and wait for its state to be 'running'
type stepBootInstance struct{}

func (s *stepBootInstance) Run(state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)

	ui.Say("Booting instance...")
	err := api.InstanceStart(client, instanceId, api.DEFAULT_TIMEOUT)
	if err != nil {
		err := fmt.Errorf("Error booting instance: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepBootInstance) Cleanup(state multistep.StateBag) {
	client := state.Get("client").(*hypercloud.ApiClient)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)
	instance, err := api.InstanceInfo(client, instanceId)
	if err != nil && instance["state"] == "running" {
		api.InstanceStop(client, instanceId, api.DEFAULT_TIMEOUT)
	}
}
