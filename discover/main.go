package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"

	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/oleksandr/bonjour"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.options", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context         *zmq.Context
	inPort, outPort *zmq.Socket
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
	context, err = zmq.NewContext()
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort(context, *inputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	context.Close()
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
	var options *bonjour.ServiceRecord
	for {
		ip, err := inPort.RecvMultipart(0)
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

	log.Println("Started...")
	go func(context *zmq.Context, endpoint string, entries chan *bonjour.ServiceEntry) {
		outPort, err = utils.CreateOutputPort(context, endpoint)
		utils.AssertError(err)
		defer outPort.Close()

		for e := range entries {
			data, err := json.Marshal(e)
			if err != nil {
				log.Println("Error encoding entry:", err.Error())
				continue
			}
			outPort.SendMultipart(runtime.NewPacket(data), 0)
		}

	}(context, *outputEndpoint, entries)

	err = bonjour.Browse(options.Service, options.Domain, entries, nil)
	if err != nil {
		ch <- syscall.SIGTERM
	}

}
