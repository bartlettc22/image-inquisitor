package sources

// RegistryLatestSemverSource is a source of images from a registry and latest semver tag
type RegistryLatestSemverSource struct {
	Tag string `yaml:"tag" json:"tag"`
}
