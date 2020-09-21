// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	servicemeshv1alpha1 "istio.io/istio/pkg/servicemesh/apis/servicemesh/v1alpha1"
	versioned "istio.io/istio/pkg/servicemesh/client/v1alpha1/clientset/versioned"
	internalinterfaces "istio.io/istio/pkg/servicemesh/client/v1alpha1/informers/externalversions/internalinterfaces"
	v1alpha1 "istio.io/istio/pkg/servicemesh/client/v1alpha1/listers/servicemesh/v1alpha1"
)

// ServiceMeshExtensionInformer provides access to a shared informer and lister for
// ServiceMeshExtensions.
type ServiceMeshExtensionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ServiceMeshExtensionLister
}

type serviceMeshExtensionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewServiceMeshExtensionInformer constructs a new informer for ServiceMeshExtension type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewServiceMeshExtensionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredServiceMeshExtensionInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredServiceMeshExtensionInformer constructs a new informer for ServiceMeshExtension type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredServiceMeshExtensionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServicemeshV1alpha1().ServiceMeshExtensions(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServicemeshV1alpha1().ServiceMeshExtensions(namespace).Watch(context.TODO(), options)
			},
		},
		&servicemeshv1alpha1.ServiceMeshExtension{},
		resyncPeriod,
		indexers,
	)
}

func (f *serviceMeshExtensionInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredServiceMeshExtensionInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *serviceMeshExtensionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&servicemeshv1alpha1.ServiceMeshExtension{}, f.defaultInformer)
}

func (f *serviceMeshExtensionInformer) Lister() v1alpha1.ServiceMeshExtensionLister {
	return v1alpha1.NewServiceMeshExtensionLister(f.Informer().GetIndexer())
}
