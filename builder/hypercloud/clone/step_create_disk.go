package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
	"sort"
)

// Clone the target disk from the template
type stepCreateDisk struct{}

type ByVersionDesc [](map[string]interface{})
func (s ByVersionDesc) Len() int {
	return len(s)
}
func (s ByVersionDesc) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByVersionDesc) Less(i, j int) bool {
	return s[i]["version"].(float64) > s[j]["version"].(float64)
}

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
	sort.Sort(ByVersionDesc(templates))
	var template map[string]interface{}
	for i := range templates {
		t := templates[i]
		templateRegionId := t["region"].(map[string]interface{})["id"].(string)
		if config.regionId == templateRegionId && (
			config.TemplateID == t["id"] ||
				(config.TemplateSlug != "" && config.TemplateSlug == t["slug"]) ||
					(config.TemplateName != "" && config.TemplateName == t["name"])) {
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

	diskName := "Packer in-progress: " + config.PackerBuildName
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
