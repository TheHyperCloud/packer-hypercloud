package api

import (
	"errors"
	"fmt"
	"time"

	"github.com/thehypercloud/apiclient-go"
)

var DEFAULT_TIMEOUT uint = 180

func DiskInfo(api *hypercloud.ApiClient, diskid string) (disk map[string]interface{}, err error) {
	status, disk, error := api.Disk.Show(diskid)
	if diskid == "" {
		return nil, fmt.Errorf("diskid cannot be blank for disk info")
	}
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return disk, errors.New(fmt.Sprint(status))
	}
	return disk, nil
}

func DiskList(api *hypercloud.ApiClient) (disk []map[string]interface{}, err error) {
	status, disks, error := api.Disk.List()
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return nil, errors.New(fmt.Sprint(disks))
	}
	return disks, nil
}

func UpdateDisk(api *hypercloud.ApiClient, diskid string, params map[string]interface{}) (disk map[string]interface{}, err error) {
	status, disk, err := api.Disk.Update(diskid, params)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return disk, errors.New(fmt.Sprintf("%d", status))
	}
	return disk, nil
}

func CreateDisk(api *hypercloud.ApiClient, data map[string]interface{}) (disk map[string]interface{}, err error)  {
	status, disk, error := api.Disk.Create(data)

	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return disk, errors.New(fmt.Sprintf("%d: %s", status, disk))
	}

	for { //ever
		disk, err = DiskInfo(api, disk["id"].(string))
		if err != nil {
			return nil, err
		} else if disk["state"] == "unattached" {
			return disk, nil
		}
		time.Sleep(2)
	}

	return disk, nil
}

func CreateBlankDisk(api *hypercloud.ApiClient, size uint, name string, region string, tier string) (disk map[string]interface{}, err error) {
	return CreateDisk(api, map[string]interface{}{
		"name":             name,
		"size":             size,
		"region":           region,
		"performance_tier": tier,
	})
}

func CreateTemplateDisk(api *hypercloud.ApiClient, size uint, name string, region string, tier string, template string) (disk map[string]interface{}, err error) {
	return CreateDisk(api, map[string]interface{}{
		"name":             name,
		"size":             size,
		"region":           region,
		"performance_tier": tier,
		"template":         template,
	})
}

func DiskDelete(api *hypercloud.ApiClient, diskid string) (err error) {
	status, disk, err := api.Disk.Delete(diskid)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return errors.New(fmt.Sprintf("Error deleting disk: %d: %s", status, disk))
	}
	return nil
}
