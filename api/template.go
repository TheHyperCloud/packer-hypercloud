package api

import (
	"errors"
	"fmt"
	"github.com/thehypercloud/apiclient-go"
)

func ListTemplates(api *hypercloud.ApiClient) (templates []map[string]interface{}, err error) {
	status, templates, error := api.Template.List()
	if error != nil {
		return nil, error
	}
	if status < 200 || status >= 300 {
		return templates, errors.New(fmt.Sprint(status))
	}

	return templates, nil
}
