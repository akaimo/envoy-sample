package main

import (
	"fmt"
	"net"
	"os"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	lsn, err := net.Listen("tcp", "0.0.0.0:20000")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
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
	"nginx_cluster": {{"nginx1", 80}, {"nginx2", 80}},
	"httpd_cluster": {{"httpd1", 80}, {"httpd2", 80}},
}

func defaultSnapshot() cache.Snapshot {
	var resources []cache.Resource
	for cluster, ups := range upstreams {
		eps := make([]*endpoint.LocalityLbEndpoints, len(ups))
		for i, up := range ups {
			addr, err := net.ResolveIPAddr("ip", up.Address)
			if err != nil {
				fmt.Println("Resolve error ", err.Error())
				os.Exit(1)
			}
			
			eps[i] = &endpoint.LocalityLbEndpoints{
				LbEndpoints: []*endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint{
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Address:       addr.IP.String(),
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
