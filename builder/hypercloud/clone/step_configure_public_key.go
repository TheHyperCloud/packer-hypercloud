package clone

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
	"os"
	"io/ioutil"
	"strings"
	"path/filepath"
)

type stepConfigurePublicKey struct{}

func (s *stepConfigurePublicKey) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*hypercloud.ApiClient)
	ui := state.Get("ui").(packer.Ui)
	instance := state.Get("instance").(map[string]interface{})
	instanceId := instance["id"].(string)

	pubKeyPath := config.Comm.SSHPrivateKey + ".pub"
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		err := fmt.Errorf("SSH public key file does not exist: %s", pubKeyPath)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	publicKeyData, err := ioutil.ReadFile(pubKeyPath); if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	publicKeyContents := strings.TrimSpace(string(publicKeyData))

	keys, err := api.ListPublicKeys(client); if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	var publicKey map[string]interface{}
	for i := range keys {
		key := keys[i]
		if strings.TrimSpace(key["key"].(string)) == publicKeyContents {
			ui.Say("Public key already in system (matched by key content)")
			publicKey = key
			break
		}
	}
	if publicKey == nil {
		ui.Say("Public key not found. Creating.")
		publicKey, err = api.PublicKeyCreate(client, "packer-" + filepath.Base(config.Comm.SSHPrivateKey), publicKeyContents); if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	publicKeyId := publicKey["id"].(string)
	err = api.InstanceUpdatePublicKeys(client, instanceId, []string{publicKeyId}); if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepConfigurePublicKey) Cleanup(state multistep.StateBag) {}
