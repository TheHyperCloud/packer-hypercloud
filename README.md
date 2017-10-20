# packer-hypercloud
This is a plugin for [Packer](https://github.com/hashicorp/packer) which works with the HyperCloud platform.

## Building from source
* At least Go 1.5.3 is required, as well as govendor (go get -u github.com/kardianos/govendor)
* ```go get -d github.com/thehypercloud/packer-hypercloud```
* ```cd "$GOPATH"/src/github.com/thehypercloud/packer-hypercloud```
* ```./install.sh```

The plugins will be installed to your GOBIN and should now be available for packer to use.

## Usage

### hypercloud-clone
This plugin creates a new boot disk from the given template id, 
boots up an instance with it, and then allows you to execute provision steps over SSH.

#### Configuration Reference
* Note: Either hypercloud_access_token OR BOTH hypercloud_id AND hypercloud_secret are required.
* Note: Either template_id OR template_name is required

##### Required
|setting|type|description|
|-------|----|-----------|
|hypercloud_url|string|Base URL of HyperCloud compatible system. e.g. 'https://my.cloud.example.net'|
|disk_performance_tier_id|string|ID of disk performance tier used to create the boot disk|
|instance_performance_tier_id|string|ID of instance performance tier used for builder instance|
|network_id|string|ID of network that will be attached to the builder instance. Should provide a connection to the internet if provisioning steps will include updating from repos etc.|
|ssh_username|string|SSH username used to connect to instance|
|ssh_private_key_file|string|Path to ssh private key file used to authenticate with instance|

##### Optional
|setting|type|description|
|-------|----|-----------|
|template_name|string|Name of template to create disk from|
|template_id|string|ID of template to create disk from|
|vm_name|string|Name of the instance, also used to name the finished disk|
|memory|integer|RAM in megabytes of the builder instance. Defaults to 512|
|ssh_wait_timeout|string|Time to wait for SSH to connect, in Go duration strings e.g. '45m'|

### hypercloud-vnc
This plugin is intended to create images /from scratch/ i.e. starting from a blank disk.

This is accomplished by booting from an install ISO media (like the Ubuntu installer CD) and 
installing the operating system onto the blank disk.

The plugin connects to the instance over VNC and types a boot command, which can specify 
how to install the OS in an unattended fashion by using some kind of preseed file,
delivered over HTTP. For this reason, the 'builder' instance running needs to be able to 
communicate back to the machine running the packer builder. So packer should be run
either inside the same private network as the target instance, or some kind of routing 
VPN needs to be set up.

#### Configuration Reference
Note: Either hypercloud_access_token or BOTH hypercloud_id AND hypercloud_secret are required.

##### Required
|setting|type|description|
|-------|----|-----------|
|hypercloud_url|string|Base URL of HyperCloud compatible system. e.g. 'https://my.cloud.example.net'|
|disk_performance_tier_id|string|ID of disk performance tier used to create the new blank disk|
|instance_performance_tier_id|string|ID of instance performance tier used for worker instance|
|network_id|string|ID of network that will be attached to the builder instance. Should provide a connection to the internet if provisioning steps will include updating from repos etc.|
|downloader_vm_id|string|ID of instance that can be used to download ISOs not already cached. Preferrably running a standard Ubuntu or Debian distribution - requires wget|
|boot_disk_url|string|URL of ISO file to download|
|boot_disk_md5|string|md5 hash of ISO contents|
|http_ip|string|IP address of the machine running packer. Used to connect back for preseed files over HTTP|
|ssh_username|string|SSH Username used to connect to newly installed instance and the downloader instance|
|ssh_password|string|SSH password used to connect to newly installed instance|
|ssh_private_key_file|string|Path to ssh private key file used to authenticate with downloader VM|
|boot_command|array&lt;string&gt;|Command passed over VNC via simulated keyboard to start the unattended install process|
|http_directory|string|Directory used to serve files over HTTP|

##### Optional
|setting|type|description|
|-------|----|-----------|
|hypercloud_id|string|ID of application used to authenticate|
|hypercloud_secret|string|Secret used with ID to authenticate|
|hypercloud_access_token|string|Access token used to authenticate|
|vm_name|string|Name of the instance, also used to name the finished disk|
|memory|integer|RAM in megabytes of the builder instance. Defaults to 512|
|ssh_wait_timeout|string|Time to wait for SSH to connect, in Go duration strings e.g. '45m'|
