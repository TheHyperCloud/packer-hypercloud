package clone

import (
	"log"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
	"time"
)

// Shutdown the instance via the API
type stepShutdown struct{}

func (s *stepShutdown) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)

	instance, err := api.InstanceInfo(client, instanceId)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if instance["state"] == "running" {
		if config.ShutdownFromAPI {
			ui.Say("Shutting down via the API")
			if err := api.InstanceStop(client, instanceId, api.DEFAULT_TIMEOUT); err != nil {
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
		} else {
			ui.Say("Waiting for instance to shutdown...")
			for instance["state"] != "stopped" {
				time.Sleep(10*time.Second)
				instance, err = api.InstanceInfo(client, instanceId)
				if err != nil {
					ui.Error(err.Error())
				}
			}
		}
	}

	log.Println("VM shut down.")
	return multistep.ActionContinue
}

func (s *stepShutdown) Cleanup(state multistep.StateBag) {}
