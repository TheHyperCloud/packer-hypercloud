package api

import (
	"errors"
	"fmt"
	"github.com/thehypercloud/apiclient-go"
)

func FindDiskTier(api *hypercloud.ApiClient, id string) (info map[string]interface{}, err error) {
	status, tiers, error := api.PerformanceTier.List_disk()
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return info, errors.New(fmt.Sprint(status))
	}

	for i := range tiers {
		tier := tiers[i]
		if tier["id"] == id {
			return tier, nil
		}
	}
	return nil, fmt.Errorf("No disk tier with id %s was found", id)
}
