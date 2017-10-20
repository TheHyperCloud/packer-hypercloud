package vnc

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/communicator/ssh"
	"github.com/hashicorp/packer/packer"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

// Makes sure the boot disk is unattached and cdrom:true
type stepPrepareBootDisk struct{}

func (s *stepPrepareBootDisk) Run(state multistep.StateBag) multistep.StepAction {
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

	ui.Say("Preparing boot disk")
	disks, err := api.DiskList(client)
	var boot_disk map[string]interface{}

	md5_substr := "md5=" + config.BootDiskMD5
	for _, element := range disks {
		name := element["name"].(string)
		region := element["region"].(map[string]interface{})
		if strings.Contains(name, md5_substr) && region["id"] == config.regionId {
			boot_disk = element
			break
		}
	}
	if boot_disk != nil {
		ui.Say(fmt.Sprintf("Found boot disk with md5 in name: %s", boot_disk["id"]))
	} else {
		// Otherwise, we need to download from the supplied URL

		// Sanity check that the URL works
		ui.Say(fmt.Sprintf("Sanity checking the boot_disk_url exists: %s", config.BootDiskURL))
		resp, err := http.Head(config.BootDiskURL)
		if err != nil {
			err := fmt.Errorf("Error checking the supplied boot disk URL: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
		if resp.StatusCode != 200 {
			err := fmt.Errorf("Error checking the supplied boot disk URL: Status code returned was %d, expected 200", resp.StatusCode)
			state.Put("error", err)
			return multistep.ActionHalt
		}
		size_header := resp.Header.Get("Content-Length")
		if size_header == "" {
			err := fmt.Errorf("Error checking the supplied boot_disk_url: No Content-Length header was present")
			state.Put("error", err)
			return multistep.ActionHalt
		}
		content_length, err := strconv.Atoi(size_header)
		if err != nil {
			err := fmt.Errorf("Error checking the supplied boot_disk_url: Content-Length header was not an integer: %s", size_header)
			state.Put("error", err)
			return multistep.ActionHalt
		}
		if content_length < (10 * 1024 * 1024) { // if smaller than 10 megabytes - arbitrary size that probably indicates not an ISO
			err := fmt.Errorf("Error checking the supplied boot_disk_url: Content-Length is less than 10 MB - probably not an ISO")
			state.Put("error", err)
			return multistep.ActionHalt
		}

		// OK, the URL is probably fine.

		// Find our downloader VM
		ui.Say(fmt.Sprintf("Checking for downloader VM with ID: %s", config.DownloaderVMID))
		downloader_vm, err := api.InstanceInfo(client, config.DownloaderVMID)
		if err != nil {
			err := fmt.Errorf("Error getting info for Download VM: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}

		// Create the new disk
		ui.Say("Creating blank disk to be used as the boot disk")
		boot_disk, err = api.CreateBlankDisk(client, 10, "Downloading... "+config.PackerBuildName, config.regionId, config.DiskPerformanceTierID)
		if err != nil {
			err := fmt.Errorf("Error creating new blank disk for boot disk via api: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}

		// Attach to downloader VM
		ui.Say("Attaching the new disk to the downloader VM")
		err = api.InstanceAddDisk(client, config.DownloaderVMID, boot_disk["id"].(string))
		if err != nil {
			err := fmt.Errorf("Error attaching new boot disk to downloader VM: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
		// Boot the downloader if not already running
		if downloader_vm["state"].(string) == "stopped" {
			ui.Say("Booting downloader vm")
			err := api.InstanceStart(client, downloader_vm["id"].(string), api.DEFAULT_TIMEOUT)
			if err != nil {
				err := fmt.Errorf("Error starting download vm: %s", err)
				state.Put("error", err)
				return multistep.ActionHalt
			}
			time.Sleep(30)
		}
		// Get the disk position in the downloader VM
		downloader_vm, err = api.InstanceInfo(client, config.DownloaderVMID)
		if err != nil {
			err := fmt.Errorf("Error getting info for Download VM: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
		instance_disks := downloader_vm["disks"].([]interface{})
		boot_disk_index := -1
		for _, element := range instance_disks {
			current_disk := element.(map[string]interface{})
			if current_disk["id"] == boot_disk["id"] {
				boot_disk_index = int(current_disk["position"].(float64))
				break
			}
		}
		if boot_disk_index == -1 {
			err := fmt.Errorf("Couldn't find index of disk %s attached to instance %s", boot_disk["id"], downloader_vm["id"])
			state.Put("error", err)
			return multistep.ActionHalt
		}

		// Just get the first IP address of the downloader VM
		adapters := downloader_vm["network_adapters"].([]interface{})
		first_adapter := adapters[0].(map[string]interface{})
		ips := first_adapter["ip_addresses"].([]interface{})
		first_ip := ips[0].(map[string]interface{})
		ip_address := first_ip["address"].(string)
		ssh_address := ip_address + ":22"

		// Turn a disk position into device path, e.g. position 1 = /dev/xvdb
		target_device := fmt.Sprintf("/dev/xvd%c", 97+boot_disk_index) // asccii char: starting at 'a' for 0 index

		// Connect to downloader VM over SSH
		ssh_config, err := sshConfig(state)
		connFunc := ssh.ConnectFunc("tcp", ssh_address)
		nc, err := connFunc()
		if err != nil {
			err = fmt.Errorf("Connecting to Downloader VM failed: TCP connection to SSH ip/port failed: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
		nc.Close()

		// Then we attempt to connect via SSH
		ssh_connection := &ssh.Config{
			Connection: connFunc,
			SSHConfig:  ssh_config,
			Pty:        true,
		}
		comm, err := ssh.New(ssh_address, ssh_connection)
		if err != nil {
			err = fmt.Errorf("Connecting to Downloader VM failed: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}

		ui.Say("Running SSH command to download file and dd to boot disk")
		command := fmt.Sprintf("wget %s -qO- > %s && if [ $(dd if=%s | head -c %d | md5sum | cut -d ' ' -f1) != \"%s\" ]; then echo 'md5 does not match'; exit 111; fi",
			config.BootDiskURL, target_device, target_device, content_length, config.BootDiskMD5)
		ui.Say(command)
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		remoteCmd := &packer.RemoteCmd{
			Command: command,
			Stdout:  stdout,
			Stderr:  stderr,
		}
		err = comm.Start(remoteCmd)
		if err != nil {
			err = fmt.Errorf("Error starting download command: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}

		remoteCmd.Wait()

		if remoteCmd.ExitStatus != 0 {
			err = fmt.Errorf("Got exit status %d from downloader SSH command, expected 0. Stdout: %s. Stderr: %s", remoteCmd.ExitStatus, stdout.String(), stderr.String())
			state.Put("error", err)
			return multistep.ActionHalt
		}

		// Rename the disk to the builder name, and include the MD5 hash
		boot_disk, err = api.UpdateDisk(client, boot_disk["id"].(string), map[string]interface{}{
			"name": config.PackerBuildName + " " + md5_substr,
		})
		if err != nil {
			err = fmt.Errorf("Error renaming boot_disk: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}

		// Live detach the boot disk from downloader VM
		err = api.InstanceRemoveDisk(client, downloader_vm["id"].(string), boot_disk["id"].(string))
		if err != nil {
			err = fmt.Errorf("Error live detaching boot disk from instance: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
	}

	// This step continues either from downloading the disk, or it already being ready
	// Get up-to-date information on the boot_disk, referred to as 'disk' from here
	disk, err := api.DiskInfo(client, boot_disk["id"].(string))
	if err != nil {
		err := fmt.Errorf("Error preparing boot disk: couldn't get disk info: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Detach from a VM if already attached
	if disk["instance_id"] != nil {
		instance_id := disk["instance_id"].(string)
		ui.Say(fmt.Sprintf("Disk is attached to instance %s", instance_id))
		instance, err := api.InstanceInfo(client, instance_id)
		if err != nil {
			err := fmt.Errorf("Error preparing boot disk: couldn't get info about instance it is already attached to: %s", err, instance)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		if instance["state"] == "stopped" {
			ui.Say("instance is stopped, doing a quick non-live disk detach")
			err = api.InstanceRemoveDisk(client, instance_id, disk["id"].(string))
		} else {
			err := fmt.Errorf("Error preparing boot disk: already attached to an instance %s in state %s", disk["instance_id"], instance["state"])
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Set disk.cdrom = true
	disk, err = api.UpdateDisk(client, disk["id"].(string), map[string]interface{}{
		"cdrom": true,
	})
	if err != nil {
		err := fmt.Errorf("Error setting cdrom:true: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("boot_disk", disk)
	return multistep.ActionContinue
}

func (s *stepPrepareBootDisk) Cleanup(state multistep.StateBag) {
	// No cleanup, disk can be re-used later
	// TODO: remove the disk if it's still got 'Downloading' in the name
}
