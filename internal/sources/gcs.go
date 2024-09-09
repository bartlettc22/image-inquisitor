package sources

type GCSSource struct {
	*GCSSourceConfig
}

type GCSSourceConfig struct {
}

func NewGCSSource(config *GCSSourceConfig) *GCSSource {
	return &GCSSource{
		GCSSourceConfig: config,
	}
}
