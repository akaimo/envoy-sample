package main

import (
	"fmt"
	"net"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"google.golang.org/grpc"
)

func main() {
	err := run()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

func run() error {
	snapshotCache := cache.NewSnapshotCache(false, hash{}, nil)
	server := xds.NewServer(snapshotCache, nil)

		err := snapshotCache.SetSnapshot("cluster.local/node0", defaultSnapshot())
		if err != nil {
			return err
		}

	grpcServer := grpc.NewServer()
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)

	lsn, err := net.Listen("tcp", "0.0.0.0:80")
	if err != nil {
		return err
	}
	fmt.Println("hoge")
	return grpcServer.Serve(lsn)
}

type hash struct{}

func (hash) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Cluster + "/" + node.Id
}

var upstreams = map[string][]struct {
	Address string
	Port    uint32
}{
	"service_nginx": {{"nginx1", 80}, {"nginx2", 80}},
	"service_httpd": {{"httpd1", 80}, {"httpd2", 80}},
}

func defaultSnapshot() cache.Snapshot {
	var resources []cache.Resource
	for cluster, ups := range upstreams {
		eps := make([]*endpoint.LocalityLbEndpoints, len(ups))
		for i, up := range ups {
			eps[i] = &endpoint.LocalityLbEndpoints{
				LbEndpoints: []*endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint{
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Address:       up.Address,
										PortSpecifier: &core.SocketAddress_PortValue{PortValue: up.Port},
									},
								},
							},
						},
					},
				}},
			}
		}
		assignment := &api.ClusterLoadAssignment{
			ClusterName: cluster,
			Endpoints:   eps,
		}
		resources = append(resources, assignment)
	}

	return cache.NewSnapshot("0.0", resources, nil, nil, nil)
}
