package vnc

import (
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// Boot the instance and wait for its state to be 'running'
type stepDisableCDBoot struct{}

func (s *stepDisableCDBoot) Run(state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)

	instance, err := api.InstanceUpdate(client, instanceId, map[string]interface{}{
		"boot_device": "disk",
	})
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("instance", instance)
	return multistep.ActionContinue
}

func (s *stepDisableCDBoot) Cleanup(state multistep.StateBag) {}
