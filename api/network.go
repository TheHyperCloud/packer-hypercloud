package api

import (
	"errors"
	"fmt"

	"github.com/thehypercloud/apiclient-go"
)

func NetworkInfo(api *hypercloud.ApiClient, id string) (info map[string]interface{}, err error) {
	status, info, error := api.Network.Show(id)
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return info, errors.New(fmt.Sprint(status))
	}
	return info, nil
}

func AllocateIP(api *hypercloud.ApiClient, networkid string, ipname string) (ip map[string]interface{}, err error) {
	args := map[string]interface{}{
		"network": networkid,
	}
	if ipname != "" {
		args["name"] = ipname
	}
	status, result, err := api.IpAddress.Allocate(args)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return result, errors.New(fmt.Sprint(status))
	}
	return result, nil
}

func DeallocateIP(api *hypercloud.ApiClient, ipId string) (err error) {
	status, _, err := api.IpAddress.Deallocate(ipId)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return errors.New(fmt.Sprint(status))
	}
	return nil
}
