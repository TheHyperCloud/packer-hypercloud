package api

import (
	"fmt"
	"time"

	"github.com/thehypercloud/apiclient-go"
)

func InstanceInfo(api *hypercloud.ApiClient, instanceId string) (disk map[string]interface{}, err error) {
	status, disk, error := api.Instance.Show(instanceId)
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return disk, fmt.Errorf("%d : %s", status, disk)
	}
	return disk, nil
}

func InstanceCreate(api *hypercloud.ApiClient, name string, memory uint, tier string, region string, diskids []string, ipids []string, boot_device string) (instance map[string]interface{}, err error) {
	args := map[string]interface{}{
		"name":              name,
		"memory":            memory,
		"performance_tier":  tier,
		"region":            region,
		"boot_device":       boot_device,
		"disks":             diskids,
		"ip_addresses":      ipids,
		"virtualization":    "hvm",
		"start_on_shutdown": false,
		"start_on_reboot":   true,
		"start_on_crash":    false,
	}

	status, result, err := api.Instance.Create_advanced(args)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return result, fmt.Errorf("%d : %s", status, result)
	}
	return result, nil
}

func InstanceUpdate(api *hypercloud.ApiClient, instanceid string, data map[string]interface{}) (instance map[string]interface{}, err error) {
	status, result, err := api.Instance.Update(instanceid, data)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("%d : %s", status, result)
	}
	return result, nil
}

func InstanceUpdatePublicKeys(api *hypercloud.ApiClient, instanceid string, keys []string) (error) {
	status, result, err := api.Instance.Update_public_keys(instanceid, map[string]interface{}{
		"public_keys": keys,
	})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("%d : %s", status, result)
	}
	return nil
}


func InstanceUpdateDisks(api *hypercloud.ApiClient, instanceid string, diskids []string) (err error) {
	status, _, err := api.Instance.Update_disks(instanceid, map[string]interface{}{
		"disks": diskids,
	})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("error when getting instance info during update disks: %s", status)
	}

	instance, err := InstanceInfo(api, instanceid)
	if err != nil {
		return err
	}
	if instance["state"] == "stopped" {
		return nil
	}

	// Loop until live attach/detach has happened
	for {
		instance, err := InstanceInfo(api, instanceid)
		if err != nil {
			return err
		}
		aching := 0
		for i := range instance["disks"].([]interface{}) {
			disks := instance["disks"].([]interface{})
			disk := disks[i].(map[string]interface{})
			if disk["state"] == "attaching" || disk["state"] == "detaching" {
				aching += 1
			}
		}
		if aching == 0 {
			break
		}
		time.Sleep(2) // Wait for live attach/detach
	}
	return nil
}

func InstanceRemoveDisk(api *hypercloud.ApiClient, instanceid string, diskid string) (err error) {
	var new_disk_ids [100]string
	instance, err := InstanceInfo(api, instanceid)
	if err != nil {
		return err
	}
	disks := instance["disks"].([]interface{})
	disk_counter := 0
	for i := range disks {
		disk := disks[i].(map[string]interface{})
		disk_id := disk["id"].(string)
		if disk_id != diskid {
			new_disk_ids[disk_counter] = disk["id"].(string)
			disk_counter++
		}
	}
	return InstanceUpdateDisks(api, instanceid, new_disk_ids[0:disk_counter])
}

func InstanceWaitForState(api *hypercloud.ApiClient, instanceid string, desiredState string, timeout time.Duration) (err error) {
	// Loop until correct state
	startTime := time.Now()
	for {
		status, body, err := api.Instance.State(instanceid)
		if err != nil {
			return err
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("error waiting for instance %s to be in state %s", instanceid, desiredState)
		}
		state := body["state"].(string)

		if state == desiredState {
			break
		}

		if time.Now().Sub(startTime).Seconds() > timeout.Seconds() {
			return fmt.Errorf("timeout of %s seconds exceeded while waiting for instance %s to be %s", timeout, instanceid, desiredState)
		}
		time.Sleep(2)
	}
	return nil // Successfull result, no error
}

func InstanceAddDisk(api *hypercloud.ApiClient, instanceid string, diskid string) (err error) {
	var new_disk_ids [100]string
	instance, err := InstanceInfo(api, instanceid)
	if err != nil {
		return err
	}
	disks := instance["disks"].([]interface{})
	var x int
	for i := range disks {
		disk := disks[i].(map[string]interface{})
		new_disk_ids[i] = disk["id"].(string)
		x = i
	}
	x++
	new_disk_ids[x] = diskid
	return InstanceUpdateDisks(api, instanceid, new_disk_ids[0:x+1])
}

const (
	start = "start"
	stop  = "stop"
)

func InstanceStart(api *hypercloud.ApiClient, instanceid string, timeout uint) (err error) {
	return startStop(api, instanceid, timeout, start)
}
func InstanceStop(api *hypercloud.ApiClient, instanceid string, timeout uint) (err error) {
	return startStop(api, instanceid, timeout, stop)
}

func startStop(api *hypercloud.ApiClient, instanceid string, timeout uint, action string) (err error) {
	var status int
	var instance map[string]interface{}
	if action == start {
		status, instance, err = api.Instance.Start(instanceid)
	} else if action == stop {
		status, instance, err = api.Instance.Stop(instanceid)
	} else {
		return fmt.Errorf("unknown action type %d", action)
	}
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("error when %s instance: %s", action, instance)
	}

	started := time.Now()

	var desiredState string
	if action == start {
		desiredState = "running"
	} else {
		desiredState = "stopped"
	}

	// Loop until correct state
	for {
		status, body, err := api.Instance.State(instanceid)
		if err != nil {
			return err
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("error when %s instance then getting state: %s", action, body)
		}
		state := body["state"].(string)

		if state == desiredState {
			break
		}

		if time.Now().Sub(started).Seconds() > float64(timeout) {
			return fmt.Errorf("timeout of %s seconds exceeded while waiting for instance %s to %s", timeout, instanceid, action)
		}
		time.Sleep(2)
	}
	return nil // Successfull result, no error
}

type ConsoleSession struct {
	Host        string
	Port        uint
	Url         string
	Token       string
	ConsoleType string
	InstanceID  string
}

func (session *ConsoleSession) Request(api *hypercloud.ApiClient, timeout uint) (err error) {
	status, request, err := api.Instance.Remote_access(session.InstanceID, map[string]interface{}{"type": session.ConsoleType})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("error when request console session: %s", request)
	}

	sessionId := request["id"].(string)
	started := time.Now()

	// Loop until correct state
	for {
		status, request, err := api.ConsoleSession.Show(sessionId)
		if err != nil {
			return err
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("error when polling console session info: %s", request)
		}
		state := request["state"].(string)

		if state == "ready" {
			session.ConsoleType = request["type"].(string)
			session.Token = request["token"].(string)
			if session.ConsoleType == "vnc" {
				session.Url = request["url"].(string)
			} else {
				session.Host = request["host"].(string)
				session.Port = uint(request["port"].(float64))
			}
			break
		}

		if time.Now().Sub(started).Seconds() > float64(timeout) {
			return fmt.Errorf("timeout of %s seconds exceeded while waiting for console session %s to become ready", timeout, sessionId)
		}
		time.Sleep(2)
	}
	return nil // Successfull result, no error
}

func InstanceRemoveNetworks(api *hypercloud.ApiClient, instanceid string) (err error) {
	status, body, err := api.Instance.Update_networking(instanceid, map[string]interface{}{
		"network_adapters": make([]map[string]interface{}, 0),
	})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("error when removing ips: %d: %s", status, body)
	}
	return nil
}

func InstanceTerminate(api *hypercloud.ApiClient, instanceid string, timeout uint, wait bool) (err error) {
	status, body, err := api.Instance.Delete(instanceid)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("error when deleting instance: %d: %s", status, body)
	}

	if wait {
		for {
			instance, err := InstanceInfo(api, instanceid)
			if err != nil {
				return err
			}
			state := instance["state"].(string)
			if state == "terminated" {
				break
			}
			time.Sleep(2)
		}
	}
	return nil
}
