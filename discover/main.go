package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/oleksandr/bonjour"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.options", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort, outPort *zmq.Socket
	outCh           chan bool
	resolver        *bonjour.Resolver
	err             error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	inPort, err = utils.CreateInputPort("", *inputEndpoint, nil)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("bonjour/discover.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	outPort.Close()
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

	exitCh := utils.HandleInterruption()
	outCh = make(chan bool)

	openPorts()
	defer closePorts()

	waitCh := make(chan bool)
	go func() {
		for {
			v := <-outCh
			if v && waitCh != nil {
				waitCh <- true
			}
			if !v {
				log.Println("OUT port is closed. Interrupting execution")
				exitCh <- syscall.SIGTERM
				break
			}
		}
	}()

	log.Println("Waiting for port connections to establish... ")
	select {
	case <-waitCh:
		log.Println("Output port connected")
		waitCh = nil
	case <-time.Tick(30 * time.Second):
		log.Println("Timeout: port connections were not established within provided interval")
		os.Exit(1)
	}

	log.Println("Waiting for configuration IP...")
	var options *bonjour.ServiceRecord
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

	resolver, err = bonjour.NewResolver(nil)
	utils.AssertError(err)

	entries := make(chan *bonjour.ServiceEntry)
	err = resolver.Browse(options.Service, options.Domain, entries)
	utils.AssertError(err)

	log.Println("Started...")
	for e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			log.Println("Error encoding entry:", err.Error())
			continue
		}
		outPort.SendMessage(runtime.NewPacket(data))
	}

}
