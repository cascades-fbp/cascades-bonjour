package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
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
	inPort    *zmq.Socket
	bonjourCh chan<- bool
	exitCh    chan os.Signal
	err       error
)

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

	// Communication channels
	bonjourCh = make(chan bool, 1)
	exitCh = make(chan os.Signal, 1)

	// Start the communication & processing logic
	go mainLoop()

	// Wait for the end...
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	<-exitCh

	// Shutdown registration gracefully
	bonjourCh <- true

	log.Println("Done")
}

// mainLoop initiates all ports and handles the traffic
func mainLoop() {
	openPorts()
	defer closePorts()

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
	var err error
	bonjourCh, err = bonjour.Register(options.Instance, options.Service, options.Domain, options.Port, options.Text, nil)
	if err != nil {
		log.Println("Error registering service:", err.Error())
		exitCh <- syscall.SIGTERM
		return
	}

	// Block execution
	select {}
}

// validateArgs checks all required flags
func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

// openPorts create ZMQ sockets and start socket monitoring loops
func openPorts() {
	inPort, err = utils.CreateInputPort("bonjour/discover.options", *inputEndpoint, nil)
	utils.AssertError(err)
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	log.Println("Closing ports...")
	inPort.Close()
	zmq.Term()
}
