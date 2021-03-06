package api

import (
	"container/list"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// ClusterResourceQuota mirrors ResourceQuota at a cluster scope.  This object is easily convertible to
// synthetic ResourceQuota object to allow quota evaluation re-use.
type ClusterResourceQuota struct {
	unversioned.TypeMeta
	// Standard object's metadata.
	kapi.ObjectMeta

	// Spec defines the desired quota
	Spec ClusterResourceQuotaSpec

	// Status defines the actual enforced quota and its current usage
	Status ClusterResourceQuotaStatus
}

// ClusterResourceQuotaSpec defines the desired quota restrictions
type ClusterResourceQuotaSpec struct {
	// Selector is the label selector used to match projects.  It is not allowed to be empty
	// and should only select active projects on the scale of dozens (though it can select
	// many more less active projects).  These projects will contend on object creation through
	// this resource.
	Selector *unversioned.LabelSelector

	// Quota defines the desired quota
	Quota kapi.ResourceQuotaSpec
}

// ClusterResourceQuotaStatus defines the actual enforced quota and its current usage
type ClusterResourceQuotaStatus struct {
	// Total defines the actual enforced quota and its current usage across all namespaces
	Total kapi.ResourceQuotaStatus

	// Namespaces slices the usage by namespace.  This division allows for quick resolution of
	// deletion reconcilation inside of a single namespace without requiring a recalculation
	// across all namespaces.  This map can be used to pull the deltas for a given namespace.
	Namespaces ResourceQuotasStatusByNamespace
}

// ClusterResourceQuotaList is a collection of ClusterResourceQuotas
type ClusterResourceQuotaList struct {
	unversioned.TypeMeta
	// Standard object's metadata.
	unversioned.ListMeta

	// Items is a list of ClusterResourceQuotas
	Items []ClusterResourceQuota
}

// ResourceQuotasStatusByNamespace provides type correct methods
type ResourceQuotasStatusByNamespace struct {
	orderedMap orderedMap
}

func (o *ResourceQuotasStatusByNamespace) Insert(key string, value kapi.ResourceQuotaStatus) {
	o.orderedMap.Insert(key, value)
}

func (o *ResourceQuotasStatusByNamespace) Get(key string) (kapi.ResourceQuotaStatus, bool) {
	ret, ok := o.orderedMap.Get(key)
	if !ok {
		return kapi.ResourceQuotaStatus{}, ok
	}
	return ret.(kapi.ResourceQuotaStatus), ok
}

func (o *ResourceQuotasStatusByNamespace) Remove(key string) {
	o.orderedMap.Remove(key)
}

func (o *ResourceQuotasStatusByNamespace) OrderedKeys() *list.List {
	return o.orderedMap.OrderedKeys()
}

// orderedMap is a very simple ordering a map tracking insertion order.  It allows fast and stable serializations
// for our encoding.  You could probably do something fancier with pointers to interfaces, but I didn't.
type orderedMap struct {
	backingMap  map[string]interface{}
	orderedKeys *list.List
}

// Insert puts something else in the map.  keys are ordered based on first insertion, not last touch.
func (o *orderedMap) Insert(key string, value interface{}) {
	if o.backingMap == nil {
		o.backingMap = map[string]interface{}{}
	}
	if o.orderedKeys == nil {
		o.orderedKeys = list.New()
	}

	if _, exists := o.backingMap[key]; !exists {
		o.orderedKeys.PushBack(key)
	}
	o.backingMap[key] = value
}

func (o *orderedMap) Get(key string) (interface{}, bool) {
	ret, ok := o.backingMap[key]
	return ret, ok
}

func (o *orderedMap) Remove(key string) {
	delete(o.backingMap, key)

	if o.orderedKeys == nil {
		return
	}
	for e := o.orderedKeys.Front(); e != nil; e = e.Next() {
		if e.Value.(string) == key {
			o.orderedKeys.Remove(e)
			break
		}
	}
}

// OrderedKeys returns back the ordered keys.  This can be used to build a stable serialization
func (o *orderedMap) OrderedKeys() *list.List {
	if o.orderedKeys == nil {
		o.orderedKeys = list.New()
	}
	return o.orderedKeys
}
