package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	//	"time"

	// 	"github.com/nats-io/nats.go"
	// 	"github.com/tamalsaha/nats-hop-demo/shared"
	// 	"github.com/tamalsaha/nats-hop-demo/transport"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	//	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	// crdinstall.Install(scheme)

	ctrl.SetLogger(klogr.New())
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100

	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
		//Opts: client.WarningHandlerOptions{
		//	SuppressWarnings:   false,
		//	AllowDuplicateLogs: false,
		//},
	})
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	_ = clientgoscheme.AddToScheme(scheme)

	// 	nc, err := nats.Connect(shared.NATS_URL)
	// 	if err != nil {
	// 		klog.Fatalln(err)
	// 	}
	// 	defer nc.Close()

	ctrl.SetLogger(klogr.New())
	cfg := ctrl.GetConfigOrDie()

	// 	tr, err := cfg.TransportConfig()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	cfg.Transport, err = transport.New(tr, nc, "k8s", 10000*time.Second)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return err
	}

	c, err := client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
		Opts: client.WarningHandlerOptions{
			SuppressWarnings:   false,
			AllowDuplicateLogs: false,
		},
	})
	if err != nil {
		return err
	}

	var nodes core.NodeList
	err = c.List(context.TODO(), &nodes)
	if err != nil {
		panic(err)
	}
	for _, n := range nodes.Items {
		fmt.Println(n.Name)
	}
	return nil
}

type Cachable interface {
	GVK(gvk schema.GroupVersionKind) (bool, error)
	GVR(gvr schema.GroupVersionResource) (bool, error)
}

type cachable struct {
	resources map[string][]metav1.APIResource
}

var _ Cachable = cachable{}

func NewCachable(cl discovery.DiscoveryInterface) (Cachable, error) {
	_, list, err := cl.ServerGroupsAndResources()
	if !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}
	m := map[string][]metav1.APIResource{}
	for _, rl := range list {
		m[rl.GroupVersion] = rl.APIResources
	}
	return cachable{resources: m}, nil
}

func (c cachable) GVK(gvk schema.GroupVersionKind) (bool, error) {
	rl, ok := c.resources[gvk.GroupVersion().String()]
	if !ok {
		return false, &meta.NoKindMatchError{
			GroupKind:        gvk.GroupKind(),
			SearchedVersions: []string{gvk.Version},
		}
	}
	for _, r := range rl {
		if r.Kind != gvk.Kind {
			continue
		}
		var canList, canWatch bool
		for _, verb := range r.Verbs {
			if verb == "list" {
				canList = true
			}
			if verb == "watch" {
				canWatch = true
			}
		}
		return canList && canWatch, nil
	}
	return false, &meta.NoKindMatchError{
		GroupKind:        gvk.GroupKind(),
		SearchedVersions: []string{gvk.Version},
	}
}

func (c cachable) GVR(gvr schema.GroupVersionResource) (bool, error) {
	rl, ok := c.resources[gvr.GroupVersion().String()]
	if !ok {
		return false, &meta.NoResourceMatchError{
			PartialResource: gvr,
		}
	}
	for _, r := range rl {
		if r.Name != gvr.Resource {
			continue
		}
		var canList, canWatch bool
		for _, verb := range r.Verbs {
			if verb == "list" {
				canList = true
			}
			if verb == "watch" {
				canWatch = true
			}
		}
		return canList && canWatch, nil
	}
	return false, &meta.NoResourceMatchError{
		PartialResource: gvr,
	}
}
