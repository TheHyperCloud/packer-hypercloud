package vnc

import (
	"fmt"
	"log"
	"math/rand"
	"net"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// This step finds an available port to listen on localhost for vnc proxy
// It also creates the VNC session on the actual VM
//
// Uses:
//   config *config
//   ui     packer.Ui
//
// Produces:
//   vnc_port uint - The port that VNC is configured to listen on.
type stepConfigureVNC struct{}

func (stepConfigureVNC) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	client := state.Get("client").(*hypercloud.ApiClient)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)

	// Find an available port. Note that this can still fail later on
	// because we have to release the port at some point. But this does its
	// best.
	msg := fmt.Sprintf("Looking for available port between %d and %d", config.VNCPortMin, config.VNCPortMax)
	ui.Say(msg)
	var vncPort uint
	portRange := int(config.VNCPortMax - config.VNCPortMin)
	for {
		vncPort = uint(rand.Intn(portRange)) + config.VNCPortMin
		log.Printf("Trying port: %d", vncPort)
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", vncPort))
		if err == nil {
			defer l.Close()
			break
		}
	}
	state.Put("vnc_proxy_port", vncPort)

	vncSession := api.ConsoleSession{
		ConsoleType: "vnc",
		InstanceID:  instanceId,
	}
	err := vncSession.Request(client, api.DEFAULT_TIMEOUT)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("vnc_session", vncSession)

	return multistep.ActionContinue
}

func (stepConfigureVNC) Cleanup(multistep.StateBag) {}
