// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/cmu-sei/terraform-provider-crucible/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set by the goreleaser configuration to the appropriate value for
// the compiled binary via the main.version ldflag.
var version string = "dev"

// Runs the provider
func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/cmu-sei/crucible",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
