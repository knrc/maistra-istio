//  Copyright 2018 Istio Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"

	mcp "istio.io/api/mcp/v1alpha1"
	"istio.io/istio/galley/pkg/mcp/client"

	// Import the resource package to pull in all proto types.
	_ "istio.io/istio/galley/pkg/runtime/resource"
)

var (
	serverAddr = flag.String("server", "127.0.0.1:9901", "The server address")
	types      = flag.String("types", "", "The types of resources to deploy")
	id         = flag.String("id", "", "The node id for the client")
)

type updater struct {
}

// Update interface method implementation.
func (u *updater) Update(ch *client.Change) error {
	fmt.Printf("==>\n%v\n", ch)
	return nil
}

func main() {
	flag.Parse()

	typeNames := strings.Split(*types, ",")

	u := &updater{}

	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		os.Exit(-1)
	}

	cl := mcp.NewAggregatedMeshConfigServiceClient(conn)

	c := client.New(cl, typeNames, u, *id, map[string]string{})
	c.Run(context.Background())
}
