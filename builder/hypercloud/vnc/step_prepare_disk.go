package vnc

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// This step creates the disk that will be used as the
// hard drive for the virtual machine.
type stepCreateDisk struct{}

func (s *stepCreateDisk) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)

	diskName := "Packer in-progress: " + config.PackerBuildName
	ui.Say(fmt.Sprintf("Creating blank target disk with name %s", diskName))
	disk, err := api.CreateBlankDisk(client, config.DiskSize, diskName, config.regionId, config.DiskPerformanceTierID)
	if err != nil {
		err := fmt.Errorf("Error creating target blank disk via api: %s: %s", err, disk)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	ui.Say(fmt.Sprintf("Target disk created with id: %s", disk["id"]))
	state.Put("disk", disk)
	return multistep.ActionContinue
}

func (s *stepCreateDisk) Cleanup(state multistep.StateBag) {
	// TODO: delete disk
}
