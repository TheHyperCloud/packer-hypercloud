package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// Delete the instance and IP address
type stepCleanup struct{}

func (s *stepCleanup) Run(state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)

	ip := state.Get("ip").(map[string]interface{})
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)

	ui.Say("Deleting build instance...")

	err := api.InstanceUpdateDisks(client, instanceId, make([]string, 0))
	if err != nil {
		ui.Error(fmt.Errorf("Error removing disks from instance: %s", err).Error())
	}
	ui.Say("Removing networks")
	err = api.InstanceRemoveNetworks(client, instanceId)
	if err != nil {
		ui.Error(fmt.Errorf("Error removing ips from instance: %s", err).Error())
	}
	ui.Say("Deallocating IP")
	err = api.DeallocateIP(client, ip["id"].(string))
	if err != nil {
		ui.Error(fmt.Errorf("Error deleting IP: %s", err).Error())
	}
	ui.Say("Deleting instance")
	err = api.InstanceTerminate(client, instanceId, api.DEFAULT_TIMEOUT, false)
	if err != nil {
		ui.Error(fmt.Errorf("Error deleting instance: %s", err).Error())
	}

	// Since the build actually succeeded, none of these errors are deal-breakers
	return multistep.ActionContinue
}

func (s *stepCleanup) Cleanup(state multistep.StateBag) {}
