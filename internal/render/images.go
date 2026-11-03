package render

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// Image is a container image declared by a rendered object
type Image struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	// Container is empty for an image set on a custom resource's spec (e.g. an operator's
	// spec.image), which the operator runs in containers of its own naming
	Container string `json:"container,omitempty"`
	Init      bool   `json:"init,omitempty"`
	Image     string `json:"image"`
}

func (i Image) key() string {
	return fmt.Sprintf("%s/%s/%s/%s/%t", i.Kind, i.Namespace, i.Name, i.Container, i.Init)
}

// ImageDiff is a container whose image differs between two renders.
// From is empty for an added container, To for a removed one.
type ImageDiff struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Container string `json:"container,omitempty"`
	Init      bool   `json:"init,omitempty"`
	From      string `json:"from"`
	To        string `json:"to"`
}

// builtinKinds are the core kinds whose images are only in their pod specs
var builtinKinds = map[string]bool{
	"Pod": true, "Deployment": true, "StatefulSet": true, "DaemonSet": true,
	"ReplicaSet": true, "Job": true, "CronJob": true, "ReplicationController": true,
}

// Images returns the container images of every object in a multi-document YAML stream.
// Objects without a namespace get the given one.
func Images(manifests []byte, namespace string) ([]Image, error) {
	images := []Image{}
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifests), 4096)
	for {
		obj := map[string]any{}
		if err := decoder.Decode(&obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("error parsing rendered manifests: %w", err)
		}
		images = append(images, objectImages(obj, namespace)...)
	}

	sort.Slice(images, func(i, j int) bool { return images[i].key() < images[j].key() })
	return images, nil
}

func objectImages(obj map[string]any, namespace string) []Image {
	kind, _ := obj["kind"].(string)
	if kind == "" {
		return nil
	}
	if strings.HasSuffix(kind, "List") {
		items, _ := obj["items"].([]any)
		images := []Image{}
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				images = append(images, objectImages(m, namespace)...)
			}
		}
		return images
	}

	meta, _ := obj["metadata"].(map[string]any)
	name, _ := meta["name"].(string)
	if ns, _ := meta["namespace"].(string); ns != "" {
		namespace = ns
	}

	images := []Image{}
	walkContainers(obj, func(container map[string]any, init bool) {
		cname, _ := container["name"].(string)
		image, _ := container["image"].(string)
		images = append(images, Image{Kind: kind, Name: name, Namespace: namespace, Container: cname, Init: init, Image: image})
	})

	if !builtinKinds[kind] {
		spec, _ := obj["spec"].(map[string]any)
		for _, field := range []string{"image", "imageName"} {
			if image, ok := spec[field].(string); ok && image != "" {
				images = append(images, Image{Kind: kind, Name: name, Namespace: namespace, Image: image})
			}
		}
	}

	return images
}

// walkContainers calls visit for every container (an item with an image) in any
// containers or initContainers list nested in v
func walkContainers(v any, visit func(container map[string]any, init bool)) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if list, ok := val.([]any); ok && (k == "containers" || k == "initContainers") {
				for _, item := range list {
					if c, ok := item.(map[string]any); ok {
						if _, ok := c["image"].(string); ok {
							visit(c, k == "initContainers")
						}
					}
				}
				continue
			}
			walkContainers(val, visit)
		}
	case []any:
		for _, item := range t {
			walkContainers(item, visit)
		}
	}
}

// Diff returns the containers whose image differs between two renders
func Diff(from, to []Image) []ImageDiff {
	byKey := map[string]*ImageDiff{}
	for _, i := range from {
		byKey[i.key()] = &ImageDiff{Kind: i.Kind, Name: i.Name, Namespace: i.Namespace, Container: i.Container, Init: i.Init, From: i.Image}
	}
	for _, i := range to {
		d, ok := byKey[i.key()]
		if !ok {
			d = &ImageDiff{Kind: i.Kind, Name: i.Name, Namespace: i.Namespace, Container: i.Container, Init: i.Init}
			byKey[i.key()] = d
		}
		d.To = i.Image
	}

	keys := make([]string, 0, len(byKey))
	for k, d := range byKey {
		if d.From != d.To {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	diffs := make([]ImageDiff, 0, len(keys))
	for _, k := range keys {
		diffs = append(diffs, *byKey[k])
	}
	return diffs
}
