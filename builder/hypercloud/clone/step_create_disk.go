package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// Clone the target disk from the template
type stepCreateDisk struct{}

func (s *stepCreateDisk) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)

	// Check the disk tier exists, get its region
	tier, err := api.FindDiskTier(client, config.DiskPerformanceTierID)
	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}
	region := tier["region"].(map[string]interface{})
	config.regionId = region["id"].(string)
	ui.Say(fmt.Sprintf("Disk performance tier found, in region %s", region["name"].(string)))

	templates, err := api.ListTemplates(client); if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}
	var template map[string]interface{}
	for i := range templates {
		t := templates[i]
		templateRegionId := t["region"].(map[string]interface{})["id"].(string)
		if config.regionId == templateRegionId && ( (config.TemplateName != "" && config.TemplateName == t["name"]) || t["id"] == config.TemplateID ) {
			template = t
			break
		}
	}

	if template == nil {
		err := fmt.Errorf("Could not find template: %s%s", config.TemplateID, config.TemplateName)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Creating boot disk")

	diskName := "Packer in-progress: " + config.VMName
	disk, err := api.CreateTemplateDisk(client, config.DiskSize, diskName, config.regionId, config.DiskPerformanceTierID, template["id"].(string))
	if err != nil {
		err := fmt.Errorf("Error creating template disk via api: %s: %s", err, disk)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("disk", disk)
	return multistep.ActionContinue
}

func (s *stepCreateDisk) Cleanup(state multistep.StateBag) {

}
