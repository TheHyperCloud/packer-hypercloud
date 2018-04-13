package clone

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
	"github.com/thehypercloud/apiclient-go"
	"github.com/thehypercloud/packer-hypercloud/api"
)

type Builder struct {
	config Config
	runner multistep.Runner
}

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`

	TemplateID                string `mapstructure:"template_id"`
	TemplateName              string `mapstructure:"template_name"`
	DiskPerformanceTierID     string `mapstructure:"disk_performance_tier_id"`
	InstancePerformanceTierID string `mapstructure:"instance_performance_tier_id"`
	DiskSize                  uint   `mapstructure:"disk_size"`
	NetworkID                 string `mapstructure:"network_id"`
	Memory                    uint   `mapstructure:"memory"`
	HYPERCLOUD_ID             string `mapstructure:"hypercloud_id"`
	HYPERCLOUD_SECRET         string `mapstructure:"hypercloud_secret"`
	HYPERCLOUD_URL            string `mapstructure:"hypercloud_url"`
	HYPERCLOUD_ACCESS_TOKEN   string `mapstructure:"hypercloud_access_token"`
	ShutdownFromAPI           bool   `mapstructure:"shutdown_from_api"`

	regionId       string
	virtualization string

	ctx interpolate.Context
}

func (self *Builder) Prepare(raws ...interface{}) (params []string, retErr error) {
	err := config.Decode(&self.config, &config.DecodeOpts{
		Interpolate: true,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
			},
		},
	}, raws...)

	if err != nil {
		return nil, err
	}

	var errs *packer.MultiError

	// Set defaults

	self.config.virtualization = "hvm" // We could do pv, but need to add serial support to this plugin

	if self.config.DiskSize == 0 {
		self.config.DiskSize = 10
	}

	if self.config.Memory == 0 {
		self.config.Memory = 512
	}

	if self.config.TemplateID == "" && self.config.TemplateName == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("either template_id or template_name is required"))
	}

	if self.config.HYPERCLOUD_URL == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("hypercloud_url is required"))
	}

	if self.config.HYPERCLOUD_ID != "" || self.config.HYPERCLOUD_SECRET != "" {
		if self.config.HYPERCLOUD_ID == "" {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("hypercloud_id is required when hypercloud_secret is provided"))
		} else if self.config.HYPERCLOUD_SECRET == "" {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("hypercloud_secret is required when hypercloud_id is provided"))
		}
	} else if self.config.HYPERCLOUD_ACCESS_TOKEN == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("either hypercloud_access_token or both hypercloud_id and hypercloud_secret are required"))
	}

	if self.config.DiskPerformanceTierID == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("disk_performance_tier_id is required"))
	}
	if self.config.InstancePerformanceTierID == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("instance_performance_tier_id is required"))
	}

	if self.config.NetworkID == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("network_id is required"))
	}

	if es := self.config.Comm.Prepare(&self.config.ctx); len(es) > 0 {
		errs = packer.MultiErrorAppend(errs, es...)
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}
	return nil, nil
}

func (self *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {
	var client hypercloud.ApiClient
	if self.config.HYPERCLOUD_ACCESS_TOKEN == "" {
		client = hypercloud.NewApplicationClient(self.config.HYPERCLOUD_URL, self.config.HYPERCLOUD_ID, self.config.HYPERCLOUD_SECRET)
	} else {
		client = hypercloud.NewAccessTokenClient(self.config.HYPERCLOUD_URL, self.config.HYPERCLOUD_ACCESS_TOKEN)
	}

	//Share state between the other steps using a statebag
	state := new(multistep.BasicStateBag)
	state.Put("cache", cache)
	state.Put("client", &client)
	state.Put("config", &self.config)
	state.Put("hook", hook)
	state.Put("ui", ui)

	steps := []multistep.Step{
		new(stepCreateDisk),
		new(stepAllocateIP),
		new(stepBuildInstance),
		new(stepConfigurePublicKey),
		new(stepBootInstance),
		&communicator.StepConnect{
			Config: &self.config.Comm,
			Host:   commHost,
		},
		new(common.StepProvision),
		new(stepShutdown),
		new(stepCleanup),
	}

	// Run!
	if self.config.PackerDebug {
		self.runner = &multistep.DebugRunner{
			Steps:   steps,
			PauseFn: common.MultistepDebugFn(ui),
		}
	} else {
		self.runner = &multistep.BasicRunner{Steps: steps}
	}

	self.runner.Run(state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	disk := state.Get("disk").(map[string]interface{})
	diskId := disk["id"].(string)

	// Rename the disk to signify success
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	newDiskName := fmt.Sprintf("Packer completed: %s %s", self.config.PackerBuildName, timeStr)
	disk, _ = api.UpdateDisk(&client, diskId, map[string]interface{}{
		"name": newDiskName,
	})

	artifact := &Artifact{
		diskId: diskId,
		state:  disk,
		client: &client,
	}
	return artifact, nil
}

func (self *Builder) Cancel() {
	if self.runner != nil {
		log.Println("Cancelling the step runner...")
		self.runner.Cancel()
	}
	fmt.Println("Cancelling the builder")
}
