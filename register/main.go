package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/oleksandr/bonjour"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	inputEndpoint = flag.String("port.options", "", "Component's input port endpoint")
	jsonFlag      = flag.Bool("json", false, "Print component documentation in JSON")
	debug         = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort *zmq.Socket
	err    error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	inPort, err = utils.CreateInputPort("bonjour/discover.options", *inputEndpoint, nil)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	zmq.Term()
}

func main() {
	flag.Parse()

	if *jsonFlag {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	validateArgs()

	openPorts()
	defer closePorts()

	ch := utils.HandleInterruption()

	log.Println("Waiting for configuration IP...")
	var options *bonjour.ServiceEntry
	for {
		ip, err := inPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving IP:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) || !runtime.IsPacket(ip) {
			continue
		}
		if err = json.Unmarshal(ip[1], &options); err != nil {
			log.Println("Error decoding options:", err.Error())
			continue
		}
		inPort.Close()
		break
	}

	log.Println("Started...")
	stopCh, err := bonjour.Register(options.Instance, options.Service, options.Domain, options.Port, options.Text, nil)
	if err != nil {
		log.Println("Error registering service:", err.Error())
		ch <- syscall.SIGTERM
	}

	// Block execution
	select {}

	// Shutdown registration gracefully
	stopCh <- true
}
