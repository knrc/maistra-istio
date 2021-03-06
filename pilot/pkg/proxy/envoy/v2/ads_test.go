// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v2_test

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"istio.io/istio/pilot/pkg/proxy/envoy"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/proxy/envoy/v2"
	"istio.io/istio/tests/util"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	routeA = "http.80"
	routeB = "https.443.https"
)

// Regression for envoy restart and overlapping connections
func TestAdsReconnectWithNonce(t *testing.T) {
	_ = initLocalPilotTestEnv(t)
	edsstr, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}
	err = sendEDSReq([]string{"outbound|1080||service3.default.svc.cluster.local"}, sidecarId(app3Ip, "app3"), edsstr)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := adsReceive(edsstr, 5*time.Second)

	// closes old process
	_ = edsstr.CloseSend()

	edsstr, err = connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer edsstr.CloseSend()

	err = sendEDSReqReconnect([]string{"service3.default.svc.cluster.local|http"}, edsstr, res)
	if err != nil {
		t.Fatal(err)
	}
	err = sendEDSReq([]string{"outbound|1080||service3.default.svc.cluster.local"}, sidecarId(app3Ip, "app3"), edsstr)
	if err != nil {
		t.Fatal(err)
	}
	res, _ = adsReceive(edsstr, 5*time.Second)
	_ = edsstr.CloseSend()

	t.Log("Received ", res)
}

// Regression for envoy restart and overlapping connections
func TestAdsReconnect(t *testing.T) {
	s := initLocalPilotTestEnv(t)
	edsstr, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}
	err = sendCDSReq(sidecarId(app3Ip, "app3"), edsstr)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = adsReceive(edsstr, 5*time.Second)

	// envoy restarts and reconnects
	edsstr2, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer edsstr2.CloseSend()
	err = sendCDSReq(sidecarId(app3Ip, "app3"), edsstr2)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = adsReceive(edsstr2, 5*time.Second)

	// closes old process
	_ = edsstr.CloseSend()

	time.Sleep(1 * time.Second)

	// event happens
	v2.AdsPushAll(s.EnvoyXdsServer)
	// will trigger recompute and push (we may need to make a change once diff is implemented

	m, err := adsReceive(edsstr2, 3*time.Second)
	if err != nil {
		t.Fatal("Recv failed", err)
	}
	t.Log("Received ", m)
}

func TestTLS(t *testing.T) {
	initLocalPilotTestEnv(t)
	edsstr, err := connectADSS(util.MockPilotSecureAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer edsstr.CloseSend()
	err = sendCDSReq(sidecarId(app3Ip, "app3"), edsstr)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adsReceive(edsstr, 3*time.Second)
	if err != nil {
		t.Error("Failed to receive with TLS connection ", err)
	}
}

func TestAdsClusterUpdate(t *testing.T) {
	server := initLocalPilotTestEnv(t)
	edsstr, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}

	var sendEDSReqAndVerify = func(clusterName string) {
		err = sendEDSReq([]string{clusterName}, sidecarId("1.1.1.1", "app3"), edsstr)
		if err != nil {
			t.Fatal(err)
		}
		res, err := adsReceive(edsstr, 5*time.Second)
		if err != nil {
			t.Fatal("Recv failed", err)
		}

		if res.TypeUrl != "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" {
			t.Error("Expecting type.googleapis.com/envoy.api.v2.ClusterLoadAssignment got ", res.TypeUrl)
		}
		if res.Resources[0].TypeUrl != "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" {
			t.Error("Expecting type.googleapis.com/envoy.api.v2.ClusterLoadAssignment got ", res.Resources[0].TypeUrl)
		}

		cla, err := getLoadAssignment(res)
		if err != nil {
			t.Fatal("Invalid EDS response ", err)
		}
		if cla.ClusterName != clusterName {
			t.Error(fmt.Sprintf("Expecting %s got ", clusterName), cla.ClusterName)
		}
	}

	_ = server.EnvoyXdsServer.MemRegistry.AddEndpoint("adsupdate.default.svc.cluster.local",
		"http-main", 2080, "10.2.0.1", 1080)

	cluster1 := "outbound|80||adsupdate.default.svc.cluster.local"
	sendEDSReqAndVerify(cluster1)

	// register a second endpoint
	_ = server.EnvoyXdsServer.MemRegistry.AddEndpoint("adsupdate2.default.svc.cluster.local",
		"http-status", 2080, "10.2.0.2", 1081)

	cluster2 := "outbound|80||adsupdate2.default.svc.cluster.local"
	sendEDSReqAndVerify(cluster2)
}

func TestAdsUpdate(t *testing.T) {
	server := initLocalPilotTestEnv(t)
	edsstr, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}

	// Old style cluster.
	// TODO: convert tests (except eds) to new style.
	server.EnvoyXdsServer.MemRegistry.AddService("adsupdate.default.svc.cluster.local", &model.Service{
		Hostname: "adsupdate.default.svc.cluster.local",
		Address:  "10.11.0.1",
		Ports:    testPorts(0),
	})
	_ = server.EnvoyXdsServer.MemRegistry.AddEndpoint("adsupdate.default.svc.cluster.local",
		"http-main", 2080, "10.2.0.1", 1080)

	err = sendEDSReq([]string{"outbound|2080||adsupdate.default.svc.cluster.local"}, sidecarId("1.1.1.1", "app3"), edsstr)
	if err != nil {
		t.Fatal(err)
	}

	res1, err := adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal("Recv failed", err)
	}

	if res1.TypeUrl != "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" {
		t.Error("Expecting type.googleapis.com/envoy.api.v2.ClusterLoadAssignment got ", res1.TypeUrl)
	}
	if res1.Resources[0].TypeUrl != "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" {
		t.Error("Expecting type.googleapis.com/envoy.api.v2.ClusterLoadAssignment got ", res1.Resources[0].TypeUrl)
	}
	cla, err := getLoadAssignment(res1)
	if err != nil {
		t.Fatal("Invalid EDS response ", err)
	}
	// TODO: validate VersionInfo and nonce once we settle on a scheme

	ep := cla.Endpoints
	if len(ep) == 0 {
		t.Fatal("No endpoints")
	}
	lbe := ep[0].LbEndpoints
	if len(lbe) == 0 {
		t.Fatal("No lb endpoints")
	}
	if "10.2.0.1" != lbe[0].Endpoint.Address.GetSocketAddress().Address {
		t.Error("Expecting 10.2.0.1 got ", lbe[0].Endpoint.Address.GetSocketAddress().Address)
	}
	strResponse, _ := model.ToJSONWithIndent(res1, " ")
	_ = ioutil.WriteFile(util.IstioOut+"/edsv2_sidecar.json", []byte(strResponse), 0644)

	_ = server.EnvoyXdsServer.MemRegistry.AddEndpoint("adsupdate.default.svc.cluster.local",
		"http-main", 2080, "10.1.7.1", 1080)

	// will trigger recompute and push for all clients - including some that may be closing
	// This reproduced the 'push on closed connection' bug.
	v2.AdsPushAll(server.EnvoyXdsServer)

	res1, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal("Recv2 failed", err)
	}
	strResponse, _ = model.ToJSONWithIndent(res1, " ")
	_ = ioutil.WriteFile(util.IstioOut+"/edsv2_update.json", []byte(strResponse), 0644)
	_ = edsstr.CloseSend()
}

func TestEnvoyRDSProtocolError(t *testing.T) {
	server := initLocalPilotTestEnv(t)

	edsstr, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}

	// wait for debounce
	time.Sleep(3 * envoy.DebounceAfter)

	err = sendRDSReq(gatewayId(gatewayIP), []string{routeA, routeB}, "", edsstr)
	if err != nil {
		t.Fatal(err)
	}
	res, err := adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes returned")
	}

	v2.AdsPushAll(server.EnvoyXdsServer)

	res, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Resources) != 2 {
		t.Fatal("No routes returned")
	}

	// send a protocol error
	err = sendRDSReq(gatewayId(gatewayIP), nil, res.Nonce, edsstr)
	if err != nil {
		t.Fatal(err)
	}
	// Refresh routes
	err = sendRDSReq(gatewayId(gatewayIP), []string{routeA, routeB}, "", edsstr)
	if err != nil {
		t.Fatal(err)
	}

	res, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes after protocol error")
	}
}

func TestEnvoyRDSUpdatedRouteRequest(t *testing.T) {
	server := initLocalPilotTestEnv(t)

	edsstr, err := connectADS(util.MockPilotGrpcAddr)
	if err != nil {
		t.Fatal(err)
	}

	// wait for debounce
	time.Sleep(3 * envoy.DebounceAfter)

	err = sendRDSReq(gatewayId(gatewayIP), []string{routeA}, "", edsstr)
	if err != nil {
		t.Fatal(err)
	}
	res, err := adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes returned")
	}
	route1, err := unmarshallRoute(res.Resources[0].Value)
	if err != nil || len(res.Resources) != 1 || route1.Name != routeA {
		t.Fatal("Expected only the http.80 route to be returned")
	}

	v2.AdsPushAll(server.EnvoyXdsServer)

	res, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes returned")
	}
	if len(res.Resources) != 1 {
		t.Fatal("Expected only 1 route to be returned")
	}
	route1, err = unmarshallRoute(res.Resources[0].Value)
	if err != nil || len(res.Resources) != 1 || route1.Name != routeA {
		t.Fatal("Expected only the http.80 route to be returned")
	}

	// Test update from A -> B
	err = sendRDSReq(gatewayId(gatewayIP), []string{routeB}, "", edsstr)
	if err != nil {
		t.Fatal(err)
	}
	res, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes returned")
	}
	route1, err = unmarshallRoute(res.Resources[0].Value)
	if err != nil || len(res.Resources) != 1 || route1.Name != routeB {
		t.Fatal("Expected only the http.80 route to be returned")
	}

	// Test update from B -> A, B
	err = sendRDSReq(gatewayId(gatewayIP), []string{routeA, routeB}, res.Nonce, edsstr)
	if err != nil {
		t.Fatal(err)
	}

	res, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes after protocol error")
	}
	if len(res.Resources) != 2 {
		t.Fatal("Expected 2 routes to be returned")
	}

	route1, err = unmarshallRoute(res.Resources[0].Value)
	if err != nil {
		t.Fatal(err)
	}
	route2, err := unmarshallRoute(res.Resources[1].Value)
	if err != nil {
		t.Fatal(err)
	}

	if (route1.Name == routeA && route2.Name != routeB) || (route2.Name == routeA && route1.Name != routeB) {
		t.Fatal("Expected http.80 and https.443.http routes to be returned")
	}

	// Test update from B, B -> A

	err = sendRDSReq(gatewayId(gatewayIP), []string{routeA}, "", edsstr)
	if err != nil {
		t.Fatal(err)
	}
	res, err = adsReceive(edsstr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Resources) == 0 {
		t.Fatal("No routes returned")
	}
	route1, err = unmarshallRoute(res.Resources[0].Value)
	if len(res.Resources) != 1 || route1.Name != routeA {
		t.Fatal("Expected only the http.80 route to be returned")
	}
}

func unmarshallRoute(value []byte) (*xdsapi.RouteConfiguration, error) {
	route := &xdsapi.RouteConfiguration{}
	err := route.Unmarshal(value)
	if err != nil {
		return nil, err
	}
	return route, nil
}
