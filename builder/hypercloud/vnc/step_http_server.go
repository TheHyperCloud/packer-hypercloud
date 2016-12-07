package vnc

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

// This step creates and runs the HTTP server that is serving files from the
// directory specified by the 'http_directory` configuration parameter in the
// template.
//
// Uses:
//   config *config
//   ui     packer.Ui
//
// Produces:
//   http_port int - The port the HTTP server started on.
type stepHTTPServer struct {
	l net.Listener
}

func (s *stepHTTPServer) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	var httpPort uint = 0
	if config.HTTPDir == "" {
		ui.Say("Not starting HTTP server, http_directory not set")
		state.Put("http_port", httpPort)
		return multistep.ActionContinue
	}

	// Find an available TCP port for our HTTP server
	var httpAddr string
	portRange := int(config.HTTPPortMax - config.HTTPPortMin)
	for {
		var err error
		var offset uint = 0

		if portRange > 0 {
			// Intn will panic if portRange == 0, so we do a check.
			offset = uint(rand.Intn(portRange))
		}

		httpPort = offset + config.HTTPPortMin
		httpAddr = fmt.Sprintf(":%d", httpPort) // Listen on all IPs
		log.Printf("Trying port: %d", httpPort)
		s.l, err = net.Listen("tcp", httpAddr)
		if err == nil {
			break
		}
	}

	ui.Say(fmt.Sprintf("Starting HTTP server on %s", httpAddr))

	// Start the HTTP server and run it in the background
	fileServer := http.FileServer(http.Dir(config.HTTPDir))
	server := &http.Server{Addr: httpAddr, Handler: fileServer}
	go server.Serve(s.l)

	// Save the address into the state so it can be accessed in the future
	state.Put("http_port", httpPort)

	if config.HTTPIP == "" {
		// Try and guess the IP by getting the first local IP
		ip, err := findFirstIP()
		if err != nil {
			ui.Error(fmt.Sprintf("http_ip not supplied, failed to guess the local ip: %s", err))
		} else {
			ui.Say(fmt.Sprintf("http_ip not supplied, guessing the local ip: %s", ip))
			config.HTTPIP = ip
		}
	}

	return multistep.ActionContinue
}

func (s *stepHTTPServer) Cleanup(multistep.StateBag) {
	if s.l != nil {
		// Close the listener so that the HTTP server stops
		s.l.Close()
	}
}

func findFirstIP() (ip string, err error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no local ip addresses found")
}
