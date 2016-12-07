package vnc

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mcclymont/tcp2ws-go"
	"github.com/mitchellh/go-vnc"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/template/interpolate"
	"github.com/thehypercloud/packer-hypercloud/api"
)

const KeyLeftShift uint32 = 0xFFE1

type bootCommandTemplateData struct {
	HTTPIP             string
	HTTPPort           uint
	Name               string
	HYPERCLOUD_IP      string
	HYPERCLOUD_NETMASK string
	HYPERCLOUD_CIDR    string
	HYPERCLOUD_GATEWAY string
}

// This step "types" the boot command into the VM over VNC.
//
// Uses:
//   config *config
//   http_port int
//   ui     packer.Ui
//   vnc_port uint
//
// Produces:
//   <nothing>
type stepTypeBootCommand struct{}

func (s *stepTypeBootCommand) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	httpPort := state.Get("http_port").(uint)
	ui := state.Get("ui").(packer.Ui)
	vncSession := state.Get("vnc_session").(api.ConsoleSession)
	vncProxyPort := state.Get("vnc_proxy_port").(uint)

	// Connect to VNC
	ui.Say("Connecting to VM via VNC")

	vncListen := fmt.Sprintf("127.0.0.1:%d", vncProxyPort)
	wsUrl := fmt.Sprintf("ws://%s:%d/", vncSession.Host, vncSession.Port)

	ui.Say(fmt.Sprintf("VNCProxy listening: %s", vncListen))
	ui.Say(fmt.Sprintf("Upsream WS URL: %s", wsUrl))

	go tcp2ws.Proxy(false, vncListen, wsUrl)
	time.Sleep(500 * time.Millisecond)

	nc, err := net.Dial("tcp", vncListen)
	if err != nil {
		err := fmt.Errorf("Error connecting to VNC: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer nc.Close()

	vncAuth := []vnc.ClientAuth{
		&ClientAuthVNC{Password: vncSession.Token},
	}
	c, err := vnc.Client(nc, &vnc.ClientConfig{Exclusive: false, Auth: vncAuth})
	if err != nil {
		err := fmt.Errorf("Error handshaking with VNC: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer c.Close()

	log.Printf("Connected to VNC desktop: %s", c.DesktopName)

	ctx := config.ctx
	ctx.Data = &bootCommandTemplateData{
		config.HTTPIP,
		httpPort,
		config.VMName,
		config.HYPERCLOUD_IP,
		config.HYPERCLOUD_NETMASK,
		config.HYPERCLOUD_CIDR,
		config.HYPERCLOUD_GATEWAY,
	}

	ui.Say("Typing the boot command over VNC...")
	for _, command := range config.BootCommand {
		command, err := interpolate.Render(command, &ctx)
		if err != nil {
			err := fmt.Errorf("Error preparing boot command: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Check for interrupts between typing things so we can cancel
		// since this isn't the fastest thing.
		if _, ok := state.GetOk(multistep.StateCancelled); ok {
			return multistep.ActionHalt
		}

		vncSendString(c, command)
	}

	return multistep.ActionContinue
}

func (*stepTypeBootCommand) Cleanup(multistep.StateBag) {}

func vncSendString(c *vnc.ClientConn, original string) {
	// Scancodes reference: https://github.com/qemu/qemu/blob/master/ui/vnc_keysym.h
	special := make(map[string]uint32)
	special["<bs>"] = 0xFF08
	special["<del>"] = 0xFFFF
	special["<enter>"] = 0xFF0D
	special["<esc>"] = 0xFF1B
	special["<f1>"] = 0xFFBE
	special["<f2>"] = 0xFFBF
	special["<f3>"] = 0xFFC0
	special["<f4>"] = 0xFFC1
	special["<f5>"] = 0xFFC2
	special["<f6>"] = 0xFFC3
	special["<f7>"] = 0xFFC4
	special["<f8>"] = 0xFFC5
	special["<f9>"] = 0xFFC6
	special["<f10>"] = 0xFFC7
	special["<f11>"] = 0xFFC8
	special["<f12>"] = 0xFFC9
	special["<return>"] = 0xFF0D
	special["<tab>"] = 0xFF09
	special["<up>"] = 0xFF52
	special["<down>"] = 0xFF54
	special["<left>"] = 0xFF51
	special["<right>"] = 0xFF53
	special["<spacebar>"] = 0x020
	special["<insert>"] = 0xFF63
	special["<home>"] = 0xFF50
	special["<end>"] = 0xFF57
	special["<pageUp>"] = 0xFF55
	special["<pageDown>"] = 0xFF56

	shiftedChars := "~!@#$%^&*()_+{}|:\"<>?"

	// TODO(mitchellh): Ripe for optimizations of some point, perhaps.
	for len(original) > 0 {
		var keyCode uint32
		keyShift := false

		if strings.HasPrefix(original, "<wait>") {
			log.Printf("Special code '<wait>' found, sleeping one second")
			time.Sleep(1 * time.Second)
			original = original[len("<wait>"):]
			continue
		}

		if strings.HasPrefix(original, "<wait5>") {
			log.Printf("Special code '<wait5>' found, sleeping 5 seconds")
			time.Sleep(5 * time.Second)
			original = original[len("<wait5>"):]
			continue
		}

		if strings.HasPrefix(original, "<wait10>") {
			log.Printf("Special code '<wait10>' found, sleeping 10 seconds")
			time.Sleep(10 * time.Second)
			original = original[len("<wait10>"):]
			continue
		}

		for specialCode, specialValue := range special {
			if strings.HasPrefix(original, specialCode) {
				log.Printf("Special code '%s' found, replacing with: %d", specialCode, specialValue)
				keyCode = specialValue
				original = original[len(specialCode):]
				break
			}
		}

		if keyCode == 0 {
			r, size := utf8.DecodeRuneInString(original)
			original = original[size:]
			keyCode = uint32(r)
			keyShift = unicode.IsUpper(r) || strings.ContainsRune(shiftedChars, r)

			log.Printf("Sending char '%c', code %d, shift %v", r, keyCode, keyShift)
		}

		if keyShift {
			c.KeyEvent(KeyLeftShift, true)
			time.Sleep(time.Second / 10)
		}

		c.KeyEvent(keyCode, true)
		time.Sleep(time.Second / 10)
		c.KeyEvent(keyCode, false)
		time.Sleep(time.Second / 10)

		if keyShift {
			c.KeyEvent(KeyLeftShift, false)
			time.Sleep(time.Second / 10)
		}
	}
}
