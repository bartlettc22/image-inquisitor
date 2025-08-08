package server

import "github.com/bartlettc22/image-inquisitor/internal/trivy"

type ServerConfig struct {
	ImageScanning bool
	CacheDir      string
}

type Server struct {
	Config *ServerConfig
}

func NewServer(c *ServerConfig) *Server {
	return &Server{
		Config: c,
	}
}

func (s *Server) Start() error {
	// Daemon for refreshing the trivy database
	if s.Config.ImageScanning {
		// Run the refresh job
		err := trivy.RefreshTrivyDB()
		if err != nil {
			return err
		}
	}

	return nil
	// Daemon for scanning images

	// Daemon for exporting images

	// Daemon for importing images
}
