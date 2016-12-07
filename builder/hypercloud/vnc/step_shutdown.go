package vnc

import (
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// Execute the shutdown_command over SSH and then wait for instance to stop
type stepShutdown struct{}

func (s *stepShutdown) Run(state multistep.StateBag) multistep.StepAction {
	comm := state.Get("communicator").(packer.Communicator)
	config := state.Get("config").(*Config)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)

	if config.ShutdownCommand != "" {
		ui.Say("Gracefully halting virtual machine...")
		log.Printf("Executing shutdown command: %s", config.ShutdownCommand)
		cmd := &packer.RemoteCmd{Command: config.ShutdownCommand}
		if err := cmd.StartWithUi(comm, ui); err != nil {
			err := fmt.Errorf("Failed to send shutdown command: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Start the goroutine that will time out our graceful attempt
		cancelCh := make(chan struct{}, 1)
		go func() {
			defer close(cancelCh)
			<-time.After(config.shutdownTimeout)
		}()

		log.Printf("Waiting max %s for shutdown to complete", config.shutdownTimeout)
		if err := api.InstanceWaitForState(client, instanceId, "stopped", config.shutdownTimeout); err == nil {
			return multistep.ActionContinue
		} else {
			ui.Say("Instance did not shutdown in time. Sending API shutdown message as well")
			if err := api.InstanceStop(client, instanceId, api.DEFAULT_TIMEOUT); err != nil {
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
		}
	} else {
		ui.Say("Shutting down via the API")
		if err := api.InstanceStop(client, instanceId, api.DEFAULT_TIMEOUT); err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	log.Println("VM shut down.")
	return multistep.ActionContinue
}

func (s *stepShutdown) Cleanup(state multistep.StateBag) {}
