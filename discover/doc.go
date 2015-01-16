package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Queries DNS-SD services by given service type and sends received entries to the output port",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "OPTIONS",
			Type:        "json",
			Description: "Configuration port to define discover query",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "json",
			Description: "Discovered service entries",
			Required:    true,
		},
	},
}
