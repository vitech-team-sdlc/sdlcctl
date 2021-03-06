/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1beta1 "github.com/vitech-team/sdlcctl/apis/topologyrelease/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTopologyReleases implements TopologyReleaseInterface
type FakeTopologyReleases struct {
	Fake *FakeTopologyreleaseV1beta1
	ns   string
}

var topologyreleasesResource = schema.GroupVersionResource{Group: "topologyrelease", Version: "v1beta1", Resource: "topologyreleases"}

var topologyreleasesKind = schema.GroupVersionKind{Group: "topologyrelease", Version: "v1beta1", Kind: "TopologyRelease"}

// Get takes name of the topologyRelease, and returns the corresponding topologyRelease object, and an error if there is any.
func (c *FakeTopologyReleases) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.TopologyRelease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(topologyreleasesResource, c.ns, name), &v1beta1.TopologyRelease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TopologyRelease), err
}

// List takes label and field selectors, and returns the list of TopologyReleases that match those selectors.
func (c *FakeTopologyReleases) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.TopologyReleaseList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(topologyreleasesResource, topologyreleasesKind, c.ns, opts), &v1beta1.TopologyReleaseList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.TopologyReleaseList{ListMeta: obj.(*v1beta1.TopologyReleaseList).ListMeta}
	for _, item := range obj.(*v1beta1.TopologyReleaseList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested topologyReleases.
func (c *FakeTopologyReleases) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(topologyreleasesResource, c.ns, opts))

}

// Create takes the representation of a topologyRelease and creates it.  Returns the server's representation of the topologyRelease, and an error, if there is any.
func (c *FakeTopologyReleases) Create(ctx context.Context, topologyRelease *v1beta1.TopologyRelease, opts v1.CreateOptions) (result *v1beta1.TopologyRelease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(topologyreleasesResource, c.ns, topologyRelease), &v1beta1.TopologyRelease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TopologyRelease), err
}

// Update takes the representation of a topologyRelease and updates it. Returns the server's representation of the topologyRelease, and an error, if there is any.
func (c *FakeTopologyReleases) Update(ctx context.Context, topologyRelease *v1beta1.TopologyRelease, opts v1.UpdateOptions) (result *v1beta1.TopologyRelease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(topologyreleasesResource, c.ns, topologyRelease), &v1beta1.TopologyRelease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TopologyRelease), err
}

// Delete takes name of the topologyRelease and deletes it. Returns an error if one occurs.
func (c *FakeTopologyReleases) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(topologyreleasesResource, c.ns, name), &v1beta1.TopologyRelease{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTopologyReleases) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(topologyreleasesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.TopologyReleaseList{})
	return err
}

// Patch applies the patch and returns the patched topologyRelease.
func (c *FakeTopologyReleases) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.TopologyRelease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(topologyreleasesResource, c.ns, name, pt, data, subresources...), &v1beta1.TopologyRelease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TopologyRelease), err
}
