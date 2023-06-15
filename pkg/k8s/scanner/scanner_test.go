package scanner

import (
	"context"
	"sort"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/aquasecurity/trivy-kubernetes/pkg/artifacts"
	cmd "github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/purl"
	"github.com/package-url/packageurl-go"

	"github.com/aquasecurity/trivy/pkg/flag"

	cyc "github.com/aquasecurity/trivy/pkg/sbom/cyclonedx"
	"github.com/aquasecurity/trivy/pkg/sbom/cyclonedx/core"

	"github.com/stretchr/testify/assert"
)

func TestK8sClusterInfoReport(t *testing.T) {
	flagOpts := flag.Options{ReportOptions: flag.ReportOptions{Format: "cyclonedx"}}
	tests := []struct {
		name        string
		clusterName string
		artifacts   []*artifacts.Artifact
		want        *core.Component
	}{
		{
			name:        "test cluster info with resources",
			clusterName: "test-cluster",
			artifacts: []*artifacts.Artifact{
				{
					Namespace: "kube-system",
					Kind:      "PodInfo",
					Name:      "kube-apiserver-kind-control-plane",
					RawResource: map[string]interface{}{
						"Containers": []interface{}{map[string]interface{}{
							"Digest":     "18e61c783b41758dd391ab901366ec3546b26fae00eef7e223d1f94da808e02f",
							"ID":         "kube-apiserver:v1.21.1",
							"Registry":   "k8s.gcr.io",
							"Repository": "kube-apiserver",
							"Version":    "v1.21.1",
						},
						},
						"Properties": map[string]string{
							"control_plane_components": "kube-apiserver",
						},
						"Name":      "kube-apiserver-kind-control-plane",
						"Namespace": "kube-system",
					},
				},
				{
					Kind: "NodeInfo",
					Name: "kind-control-plane",
					RawResource: map[string]interface{}{
						"ContainerRuntimeVersion": "containerd://1.5.2",
						"Hostname":                "kind-control-plane",
						"KubeProxyVersion":        "6.2.13-300.fc38.aarch64",
						"KubeletVersion":          "v1.21.1",
						"NodeName":                "kind-control-plane",
						"OsImage":                 "Ubuntu 21.04",
						"Properties": map[string]string{
							"architecture":     "arm64",
							"host_name":        "kind-control-plane",
							"kernel_version":   "6.2.15-300.fc38.aarch64",
							"node_role":        "master",
							"operating_system": "linux",
						},
					},
				},
			},
			want: &core.Component{
				Type: cdx.ComponentTypeContainer,
				Name: "test-cluster",
				Components: []*core.Component{
					{
						Type: cdx.ComponentTypeApplication,
						Name: "kube-apiserver-kind-control-plane",
						Properties: map[string]string{
							"control_plane_components": "kube-apiserver",
						},
						Components: []*core.Component{
							{
								Type:    cdx.ComponentTypeContainer,
								Name:    "k8s.gcr.io/kube-apiserver",
								Version: "sha256:18e61c783b41758dd391ab901366ec3546b26fae00eef7e223d1f94da808e02f",
								PackageURL: &purl.PackageURL{
									PackageURL: packageurl.PackageURL{
										Type:    "oci",
										Name:    "kube-apiserver",
										Version: "sha256:18e61c783b41758dd391ab901366ec3546b26fae00eef7e223d1f94da808e02f",
										Qualifiers: packageurl.Qualifiers{
											{
												Key:   "repository_url",
												Value: "k8s.gcr.io/kube-apiserver",
											},
											{
												Key: "arch",
											},
										},
									},
								},
								Properties: map[string]string{
									cyc.PropertyPkgID:   "k8s.gcr.io/kube-apiserver:1.21.1",
									cyc.PropertyPkgType: "oci",
								},
							},
						},
					},
					{
						Type: cdx.ComponentTypeContainer,
						Name: "kind-control-plane",
						Properties: map[string]string{
							"architecture":     "arm64",
							"host_name":        "kind-control-plane",
							"kernel_version":   "6.2.15-300.fc38.aarch64",
							"node_role":        "master",
							"operating_system": "linux",
						},
						Components: []*core.Component{
							{
								Type:    cdx.ComponentTypeOS,
								Name:    "ubuntu",
								Version: "21.04",
								Properties: map[string]string{
									"Class": "os-pkgs",
									"Type":  "ubuntu",
								},
							},
							{
								Type: cdx.ComponentTypeApplication,
								Name: "node-core-components",
								Properties: map[string]string{
									"Class": "lang-pkgs",
									"Type":  "golang",
								},
								Components: []*core.Component{
									{
										Type:    cdx.ComponentTypeLibrary,
										Name:    "k8s.io/kubelet",
										Version: "1.21.1",
										Properties: map[string]string{
											"PkgType": "golang",
										},
										PackageURL: &purl.PackageURL{
											PackageURL: packageurl.PackageURL{
												Type:       "golang",
												Name:       "k8s.io/kubelet",
												Version:    "1.21.1",
												Qualifiers: packageurl.Qualifiers{},
											},
										},
									},
									{
										Type:    cdx.ComponentTypeLibrary,
										Name:    "github.com/containerd/containerd",
										Version: "1.5.2",
										Properties: map[string]string{
											cyc.PropertyPkgType: "golang",
										},
										PackageURL: &purl.PackageURL{
											PackageURL: packageurl.PackageURL{
												Type:       "golang",
												Name:       "github.com/containerd/containerd",
												Version:    "1.5.2",
												Qualifiers: packageurl.Qualifiers{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			runner, err := cmd.NewRunner(ctx, flagOpts)
			assert.NoError(t, err)
			scanner := NewScanner(tt.clusterName, runner, flagOpts)
			got, err := scanner.Scan(ctx, tt.artifacts)
			sortNodeComponents(got.RootComponent)
			sortNodeComponents(tt.want)
			assert.Equal(t, tt.want, got.RootComponent)
		})
	}
}

func sortNodeComponents(component *core.Component) {
	nodeComp := findComponentByName(component, "node-core-components")
	sort.Slice(nodeComp.Components, func(i, j int) bool {
		return nodeComp.Components[i].Name < nodeComp.Components[j].Name
	})
}

func findComponentByName(component *core.Component, compName string) *core.Component {
	if component.Name == compName {
		return component
	}
	var fComp *core.Component
	for _, comp := range component.Components {
		fComp = findComponentByName(comp, compName)
	}
	return fComp
}

func TestTestOsNameVersion(t *testing.T) {
	tests := []struct {
		name        string
		nameVersion string
		compName    string
		compVersion string
	}{

		{
			name:        "valid version",
			nameVersion: "ubuntu 20.04",
			compName:    "ubuntu",
			compVersion: "20.04",
		},
		{
			name:        "valid sem version",
			nameVersion: "ubuntu 20.04.1",
			compName:    "ubuntu",
			compVersion: "20.04.1",
		},
		{
			name:        "non valid version",
			nameVersion: "ubuntu",
			compName:    "ubuntu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, version := osNameVersion(tt.nameVersion)
			assert.Equal(t, name, tt.compName)
			assert.Equal(t, version, tt.compVersion)
		})
	}
}
