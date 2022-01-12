package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	//	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	//	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	// crdinstall.Install(scheme)

	ctrl.SetLogger(klogr.New())
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100

	detector, err := NewDynamicCachable(cfg)
	if err != nil {
		panic(err)
	}
	gvk := schema.GroupVersionKind{
		Group:   "ui.k8s.appscode.com",
		Version: "v1alpha1",
		Kind:    "GenericResource",
	}

	cache, err := detector.GVK(gvk)
	if meta.IsNoMatchError(err) {
		fmt.Println(err)
	}

	cache, err = detector.GVK(gvk)
	if err != nil {
		panic(err)
	}
	fmt.Println("CACHABLE =", cache)
}

type Cachable interface {
	GVK(gvk schema.GroupVersionKind) (bool, error)
	GVR(gvr schema.GroupVersionResource) (bool, error)
}

type cachable struct {
	resources map[string][]metav1.APIResource
}

var _ Cachable = &cachable{}

func NewCachable(cl discovery.DiscoveryInterface) (Cachable, error) {
	_, list, err := cl.ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}
	m := map[string][]metav1.APIResource{}
	for _, rl := range list {
		m[rl.GroupVersion] = rl.APIResources
	}
	return &cachable{resources: m}, nil
}

func (c *cachable) GVK(gvk schema.GroupVersionKind) (bool, error) {
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

func (c *cachable) GVR(gvr schema.GroupVersionResource) (bool, error) {
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
