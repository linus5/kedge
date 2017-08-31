package discovery

import (
	"context"
	"fmt"
	"time"

	pb_config "github.com/mwitkow/kedge/_protogen/kedge/config"
	"github.com/mwitkow/kedge/lib/k8s"
	"github.com/mwitkow/kedge/lib/sharedflags"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	selectorKeySuffix                  = "kedge-exposed"
	hostMatcherAnnotationSuffix        = "host-matcher"
	serviceNameMatcherAnnotationSuffix = "service-name-matcher"
)

var (
	// TODO(bplotka): Consider moving to regex with .namespace .name variables.
	flagExternalDomainSuffix = sharedflags.Set.String("discovery_external_domain_suffix", "", "Required suffix "+
		"that will be added to service name to constructs external domain for director route")
	flagAnnotationLabelPrefix = sharedflags.Set.String("discovery_label_annotation_prefix", "kedge.com/",
		"Expected annotation/label prefix for all kedge annotations and kedge-exposed label")
)

// Routing Discovery allows to get fresh director and backendpool configuration filled with autogenerated routings based on service annotations.
// It watches every services (from whatever namespace) that have label named by 'discovery_label_annotation_prefix'/kedge-exposed.
// It goes through every service's spec port's and generates routing->backend pair.
//
// For each spec in form like:
//   Port: 1234
//	 Name: "http-something"
//   TargetPort: "pods-port"
//
// It generates:
//   host_matcher: "<service-name>.<discovery_external_domain_suffix>"
//   port_matcher: 1234
//
//   backend_name:  "<service-name>_<namespace>_pods-port"
//   domainPort for k8s lookup: "<service-name>.<namespace>:http-port"
//
//
// Similar for GRPC if name starts is "grpc" or starts from "grpc-"
//
// If you wish to override host_matcher or service_name_matcher use annotations:
//   `discovery_label_annotation_prefix`host-matcher = <domain>
//   `discovery_label_annotation_prefix`service-name-matcher = <domain>
//
// NOTE:
// - backend name is always in form of <service>_<namespace>_<port-name>
// - if no name is provided or name is not in form of grpc- or http- it is silently ignored (!)
// - TargetPort can be in both port name or port number form.
// - no check for duplicated host_matchers in annotations or between autogenerated & base ones (!)
// - no check if the target port inside service actually exists.
type RoutingDiscovery struct {
	logger                logrus.FieldLogger
	serviceClient         ServiceClient
	baseBackendpool       *pb_config.BackendPoolConfig
	baseDirector          *pb_config.DirectorConfig
	labelSelectorKey      string
	externalDomainSuffix  string
	labelAnnotationPrefix string
}

func NewFromFlags(logger logrus.FieldLogger, baseDirector *pb_config.DirectorConfig, baseBackendpool *pb_config.BackendPoolConfig) (*RoutingDiscovery, error) {
	if *flagExternalDomainSuffix == "" {
		return nil, errors.Errorf("required flag 'discovery_external_domain_suffix' is not specified.")
	}

	apiClient, err := k8s.NewFromFlags()
	if err != nil {
		return nil, err
	}
	return NewWithClient(logger, baseDirector, baseBackendpool, &client{k8sClient: apiClient}), nil
}

// NewWithClient returns a new Kubernetes RoutingDiscovery using given k8s.APIClient configured to be used against kube-apiserver.
func NewWithClient(logger logrus.FieldLogger, baseDirector *pb_config.DirectorConfig, baseBackendpool *pb_config.BackendPoolConfig, serviceClient ServiceClient) *RoutingDiscovery {
	return &RoutingDiscovery{
		logger:                logger,
		baseBackendpool:       baseBackendpool,
		baseDirector:          baseDirector,
		serviceClient:         serviceClient,
		labelSelectorKey:      fmt.Sprintf("%s%s", *flagAnnotationLabelPrefix, selectorKeySuffix),
		externalDomainSuffix:  *flagExternalDomainSuffix,
		labelAnnotationPrefix: *flagAnnotationLabelPrefix,
	}
}

// DiscoverOnce returns director & backendpool configs filled with mix of persistent routes & backends given in base configs and dynamically discovered ones.
func (d *RoutingDiscovery) DiscoverOnce(ctx context.Context) (*pb_config.DirectorConfig, *pb_config.BackendPoolConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second) // Let's give 4 seconds to gather all changes.
	defer cancel()

	watchResultCh := make(chan watchResult)
	defer close(watchResultCh)

	err := startWatchingServicesChanges(ctx, d.labelSelectorKey, d.serviceClient, watchResultCh)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Failed to start watching services by %s selector stream", d.labelSelectorKey)
	}

	updater := newUpdater(
		d.baseDirector,
		d.baseBackendpool,
		d.externalDomainSuffix,
		d.labelAnnotationPrefix,
	)

	var resultDirectorConfig *pb_config.DirectorConfig
	var resultBackendPool *pb_config.BackendPoolConfig
	for {
		var event event
		select {
		case <-ctx.Done():
			// Time is up, let's return what we have so far.
			return resultDirectorConfig, resultBackendPool, nil
		case r := <-watchResultCh:
			if r.err != nil {
				return nil, nil, errors.Wrap(r.err, "error on reading event stream")
			}
			event = *r.ep
		}

		resultDirectorConfig, resultBackendPool, err = updater.onEvent(event)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "error on updating routing on event %v", event)
		}
	}
}
