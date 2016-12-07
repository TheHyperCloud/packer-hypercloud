package clone

import (
	"fmt"

	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

const (
	builderID = "hypercloud.clone.disk"
)

type Artifact struct {
	diskId string
	state  map[string]interface{}
	client *hypercloud.ApiClient
}

func (*Artifact) BuilderId() string {
	return builderID
}

func (a *Artifact) Files() []string {
	return make([]string, 0) // empty slice - no files generated
}

func (a *Artifact) Id() string {
	return a.diskId
}

func (a *Artifact) String() string {
	name := a.State("name").(string)
	id := a.State("id").(string)
	return fmt.Sprintf("Disk: %s : %s", id, name)
}

func (a *Artifact) State(name string) interface{} {
	return a.state[name]
}

func (a *Artifact) Destroy() error {
	return api.DiskDelete(a.client, a.diskId)
}
