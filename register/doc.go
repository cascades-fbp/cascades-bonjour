package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Registers a service and announces it via DNS-SD",
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "OPTIONS",
			Type:        "json",
			Description: "Configuration port to define discover query",
			Required:    true,
		},
	},
}
