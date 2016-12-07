package api

import (
	"errors"
	"fmt"
	"github.com/thehypercloud/apiclient-go"
)

func ListPublicKeys(api *hypercloud.ApiClient) (data []map[string]interface{}, err error) {
	status, data, error := api.PublicKey.List()
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return data, errors.New(fmt.Sprint(status))
	}

	return data, nil
}

func PublicKeyCreate(api *hypercloud.ApiClient, name string, keyData string) (map[string]interface{}, error) {
	status, result, err := api.PublicKey.Create(map[string]interface{}{
		"key": keyData,
		"name": name,
	})

	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return result, fmt.Errorf("%d : %s", status, result)
	}
	return result, nil
}
