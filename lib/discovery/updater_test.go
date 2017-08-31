package discovery

import (
	"testing"

	pb_config "github.com/mwitkow/kedge/_protogen/kedge/config"
	pb_resolvers "github.com/mwitkow/kedge/_protogen/kedge/config/common/resolvers"
	pb_grpcbackends "github.com/mwitkow/kedge/_protogen/kedge/config/grpc/backends"
	pb_grpcroutes "github.com/mwitkow/kedge/_protogen/kedge/config/grpc/routes"
	pb_httpbackends "github.com/mwitkow/kedge/_protogen/kedge/config/http/backends"
	pb_httproutes "github.com/mwitkow/kedge/_protogen/kedge/config/http/routes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_OnEvent_AdditionAndModifyAndDelete_HTTP(t *testing.T) {
	updater := newUpdater(
		&pb_config.DirectorConfig{
			Grpc: &pb_config.DirectorConfig_Grpc{},
			Http: &pb_config.DirectorConfig_Http{
				Routes: []*pb_httproutes.Route{
					{
						Autogenerated: false,
						HostMatcher:   "something",
						PortMatcher:   1234,
						BackendName:   "already_there",
					},
				},
			},
		},
		&pb_config.BackendPoolConfig{
			Grpc: &pb_config.BackendPoolConfig_Grpc{},
			Http: &pb_config.BackendPoolConfig_Http{
				Backends: []*pb_httpbackends.Backend{
					{
						Name: "something",
						Resolver: &pb_httpbackends.Backend_K8S{
							K8S: &pb_resolvers.K8SResolver{
								DnsPortName: "s2.ns1:some",
							},
						},
					},
				},
			},
		},
		"some-external.example.com",
		"http.kedge-exposed.com/",
		"grpc.kedge-exposed.com/",
	)

	okEvent := event{
		Type: added,
		Object: service{
			Kind: "Services",
			Metadata: metadata{
				Name:      "s2",
				Namespace: "ns1",
				Annotations: map[string]string{
					"some-trash":                   "ok",
					"http.kedge-exposed.com/port1": "external.host.com:1",
					"http.kedge-exposed.com/port3": "external.host.com",
					"http.kedge-exposed.com/port5": "",
					"http.kedge-exposed.com/port7": ":7",
				},
			},
		},
	}

	d, b, err := updater.onEvent(okEvent)
	require.NoError(t, err)

	expectedDirectorConfig := &pb_config.DirectorConfig{
		Grpc: &pb_config.DirectorConfig_Grpc{},
		Http: &pb_config.DirectorConfig_Http{
			Routes: []*pb_httproutes.Route{
				{
					Autogenerated: false,
					HostMatcher:   "something",
					PortMatcher:   1234,
					BackendName:   "already_there",
				},
				{
					Autogenerated: true,
					HostMatcher:   "external.host.com",
					PortMatcher:   1,
					BackendName:   "s2_ns1_port1",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
				{
					Autogenerated: true,
					HostMatcher:   "external.host.com",
					PortMatcher:   0,
					BackendName:   "s2_ns1_port3",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
				{
					Autogenerated: true,
					HostMatcher:   "s2.some-external.example.com",
					PortMatcher:   7,
					BackendName:   "s2_ns1_port7",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
				{
					Autogenerated: true,
					HostMatcher:   "s2.some-external.example.com",
					PortMatcher:   0,
					BackendName:   "s2_ns1_port5",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
			},
		},
	}
	assert.Equal(t, expectedDirectorConfig, d)

	expectedBackendpoolConfig := &pb_config.BackendPoolConfig{
		Grpc: &pb_config.BackendPoolConfig_Grpc{},
		Http: &pb_config.BackendPoolConfig_Http{
			Backends: []*pb_httpbackends.Backend{
				{
					Autogenerated: true,
					Name:          "s2_ns1_port1",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port1",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port3",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port3",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port5",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port5",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port7",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port7",
						},
					},
				},
				{
					Name: "something",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:some",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expectedBackendpoolConfig, b)

	modifyEvent := event{
		Type: modified,
		Object: service{
			Kind: "Services",
			Metadata: metadata{
				Name:      "s2",
				Namespace: "ns1",
				Annotations: map[string]string{
					"some-trash":                   "ok",
					"http.kedge-exposed.com/port11": "external.host.com:11",
					"http.kedge-exposed.com/port13": "external.host.com",
					"http.kedge-exposed.com/port15": "",
					"http.kedge-exposed.com/port17": ":17",
				},
			},
		},
	}

	d, b, err = updater.onEvent(modifyEvent)
	require.NoError(t, err)

	expectedDirectorConfig2 := &pb_config.DirectorConfig{
		Grpc: &pb_config.DirectorConfig_Grpc{},
		Http: &pb_config.DirectorConfig_Http{
			Routes: []*pb_httproutes.Route{
				{
					Autogenerated: false,
					HostMatcher:   "something",
					PortMatcher:   1234,
					BackendName:   "already_there",
				},
				{
					Autogenerated: true,
					HostMatcher:   "external.host.com",
					PortMatcher:   11,
					BackendName:   "s2_ns1_port11",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
				{
					Autogenerated: true,
					HostMatcher:   "external.host.com",
					PortMatcher:   0,
					BackendName:   "s2_ns1_port13",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
				{
					Autogenerated: true,
					HostMatcher:   "s2.some-external.example.com",
					PortMatcher:   17,
					BackendName:   "s2_ns1_port17",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
				{
					Autogenerated: true,
					HostMatcher:   "s2.some-external.example.com",
					PortMatcher:   0,
					BackendName:   "s2_ns1_port15",
					ProxyMode:     pb_httproutes.ProxyMode_REVERSE_PROXY,
				},
			},
		},
	}
	assert.Equal(t, expectedDirectorConfig2, d)

	expectedBackendpoolConfig2 := &pb_config.BackendPoolConfig{
		Grpc: &pb_config.BackendPoolConfig_Grpc{},
		Http: &pb_config.BackendPoolConfig_Http{
			Backends: []*pb_httpbackends.Backend{
				{
					Autogenerated: true,
					Name:          "s2_ns1_port11",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port11",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port13",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port13",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port15",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port15",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port17",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port17",
						},
					},
				},
				{
					Name: "something",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:some",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expectedBackendpoolConfig2, b)

	deleteEvent := event{
		Type: deleted,
		Object: service{
			Kind: "Services",
			Metadata: metadata{
				Name:      "s2",
				Namespace: "ns1",
			},
		},
	}

	d, b, err = updater.onEvent(deleteEvent)
	require.NoError(t, err)

	expectedDirectorConfig3 := &pb_config.DirectorConfig{
		Grpc: &pb_config.DirectorConfig_Grpc{},
		Http: &pb_config.DirectorConfig_Http{
			Routes: []*pb_httproutes.Route{
				{
					Autogenerated: false,
					HostMatcher:   "something",
					PortMatcher:   1234,
					BackendName:   "already_there",
				},
			},
		},
	}
	assert.Equal(t, expectedDirectorConfig3, d)

	expectedBackendpoolConfig3 := &pb_config.BackendPoolConfig{
		Grpc: &pb_config.BackendPoolConfig_Grpc{},
		Http: &pb_config.BackendPoolConfig_Http{
			Backends: []*pb_httpbackends.Backend{
				{
					Name: "something",
					Resolver: &pb_httpbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:some",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expectedBackendpoolConfig3, b)
}

func TestUpdater_OnEvent_AdditionAndModifyAndDelete_GRPC(t *testing.T) {
	updater := newUpdater(
		&pb_config.DirectorConfig{
			Grpc: &pb_config.DirectorConfig_Grpc{
				Routes: []*pb_grpcroutes.Route{
					{
						Autogenerated:      false,
						ServiceNameMatcher: "something",
						PortMatcher:        1234,
						BackendName:        "already_there",
					},
				},
			},
			Http: &pb_config.DirectorConfig_Http{},
		},
		&pb_config.BackendPoolConfig{
			Grpc: &pb_config.BackendPoolConfig_Grpc{
				Backends: []*pb_grpcbackends.Backend{
					{
						Name: "something",
						Resolver: &pb_grpcbackends.Backend_K8S{
							K8S: &pb_resolvers.K8SResolver{
								DnsPortName: "s2.ns1:some-port",
							},
						},
					},
				},
			},
			Http: &pb_config.BackendPoolConfig_Http{},
		},
		"some-external.example.com",
		"http.kedge-exposed.com/",
		"grpc.kedge-exposed.com/",
	)

	okEvent := event{
		Type: added,
		Object: service{
			Metadata: metadata{
				Name:      "s2",
				Namespace: "ns1",
				Annotations: map[string]string{
					"some-trash":                   "ok",
					"grpc.kedge-exposed.com/port2": "external.com/Method1:2",
					"grpc.kedge-exposed.com/port4": "external.com/Method2",
					"grpc.kedge-exposed.com/port6": "",
					"grpc.kedge-exposed.com/port8": ":8",
				},
			},
		},
	}

	d, b, err := updater.onEvent(okEvent)
	require.NoError(t, err)

	expectedDirectorConfig := &pb_config.DirectorConfig{
		Grpc: &pb_config.DirectorConfig_Grpc{
			Routes: []*pb_grpcroutes.Route{
				{
					Autogenerated:      false,
					ServiceNameMatcher: "something",
					PortMatcher:        1234,
					BackendName:        "already_there",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "external.com/Method1",
					PortMatcher:        2,
					BackendName:        "s2_ns1_port2",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "external.com/Method2",
					PortMatcher:        0,
					BackendName:        "s2_ns1_port4",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "s2.some-external.example.com",
					PortMatcher:        8,
					BackendName:        "s2_ns1_port8",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "s2.some-external.example.com",
					PortMatcher:        0,
					BackendName:        "s2_ns1_port6",
				},
			},
		},
		Http: &pb_config.DirectorConfig_Http{},
	}
	assert.Equal(t, expectedDirectorConfig, d)

	expectedBackendpoolConfig := &pb_config.BackendPoolConfig{
		Grpc: &pb_config.BackendPoolConfig_Grpc{
			Backends: []*pb_grpcbackends.Backend{
				{
					Autogenerated: true,
					Name:          "s2_ns1_port2",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port2",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port4",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port4",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port6",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port6",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port8",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port8",
						},
					},
				},
				{
					Name: "something",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:some-port",
						},
					},
				},
			},
		},
		Http: &pb_config.BackendPoolConfig_Http{},
	}
	assert.Equal(t, expectedBackendpoolConfig, b)

	modifyEvent := event{
		Type: modified,
		Object: service{
			Metadata: metadata{
				Name:      "s2",
				Namespace: "ns1",
				Annotations: map[string]string{
					"some-trash":                   "ok",
					"grpc.kedge-exposed.com/port12": "external.com/Method11:12",
					"grpc.kedge-exposed.com/port14": "external.com/Method12",
					"grpc.kedge-exposed.com/port16": "",
					"grpc.kedge-exposed.com/port18": ":18",
				},
			},
		},
	}

	d, b, err = updater.onEvent(modifyEvent)
	require.NoError(t, err)

	expectedDirectorConfig2 := &pb_config.DirectorConfig{
		Grpc: &pb_config.DirectorConfig_Grpc{
			Routes: []*pb_grpcroutes.Route{
				{
					Autogenerated:      false,
					ServiceNameMatcher: "something",
					PortMatcher:        1234,
					BackendName:        "already_there",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "external.com/Method11",
					PortMatcher:        12,
					BackendName:        "s2_ns1_port12",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "external.com/Method12",
					PortMatcher:        0,
					BackendName:        "s2_ns1_port14",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "s2.some-external.example.com",
					PortMatcher:        18,
					BackendName:        "s2_ns1_port18",
				},
				{
					Autogenerated:      true,
					ServiceNameMatcher: "s2.some-external.example.com",
					PortMatcher:        0,
					BackendName:        "s2_ns1_port16",
				},
			},
		},
		Http: &pb_config.DirectorConfig_Http{},
	}
	assert.Equal(t, expectedDirectorConfig2, d)

	expectedBackendpoolConfig2 := &pb_config.BackendPoolConfig{
		Grpc: &pb_config.BackendPoolConfig_Grpc{
			Backends: []*pb_grpcbackends.Backend{
				{
					Autogenerated: true,
					Name:          "s2_ns1_port12",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port12",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port14",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port14",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port16",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port16",
						},
					},
				},
				{
					Autogenerated: true,
					Name:          "s2_ns1_port18",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:port18",
						},
					},
				},
				{
					Name: "something",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:some-port",
						},
					},
				},
			},
		},
		Http: &pb_config.BackendPoolConfig_Http{},
	}
	assert.Equal(t, expectedBackendpoolConfig2, b)

	deleteEvent := event{
		Type: deleted,
		Object: service{
			Kind: "Services",
			Metadata: metadata{
				Name:      "s2",
				Namespace: "ns1",
			},
		},
	}

	d, b, err = updater.onEvent(deleteEvent)
	require.NoError(t, err)

	expectedDirectorConfig3 := &pb_config.DirectorConfig{
		Grpc: &pb_config.DirectorConfig_Grpc{
			Routes: []*pb_grpcroutes.Route{
				{
					Autogenerated:      false,
					ServiceNameMatcher: "something",
					PortMatcher:        1234,
					BackendName:        "already_there",
				},
			},
		},
		Http: &pb_config.DirectorConfig_Http{},
	}
	assert.Equal(t, expectedDirectorConfig3, d)

	expectedBackendpoolConfig3 := &pb_config.BackendPoolConfig{
		Grpc: &pb_config.BackendPoolConfig_Grpc{
			Backends: []*pb_grpcbackends.Backend{
				{
					Name: "something",
					Resolver: &pb_grpcbackends.Backend_K8S{
						K8S: &pb_resolvers.K8SResolver{
							DnsPortName: "s2.ns1:some-port",
						},
					},
				},
			},
		},
		Http: &pb_config.BackendPoolConfig_Http{},
	}
	assert.Equal(t, expectedBackendpoolConfig3, b)

}
