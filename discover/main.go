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
	inputEndpoint  = flag.String("port.options", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort, outPort *zmq.Socket
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
	inPort, err = utils.CreateInputPort(*inputEndpoint)
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

	resolver, err = bonjour.NewResolver(nil)
	utils.AssertError(err)

	exitCh := utils.HandleInterruption()

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

	entries := make(chan *bonjour.ServiceEntry)

	go func(endpoint string, entries chan *bonjour.ServiceEntry, exitCh chan os.Signal) {
		outPort, err = utils.CreateOutputPort(endpoint)
		utils.AssertError(err)
		defer outPort.Close()

		for e := range entries {
			data, err := json.Marshal(e)
			if err != nil {
				log.Println("Error encoding entry:", err.Error())
				continue
			}
			outPort.SendMessage(runtime.NewPacket(data))
		}

	}(*outputEndpoint, entries, exitCh)

	err = resolver.Browse(options.Service, options.Domain, entries)
	if err != nil {
		exitCh <- syscall.SIGTERM
	} else {
		log.Println("Started...")
		select {
		case <-exitCh:
			os.Exit(0)
		}
	}

}
