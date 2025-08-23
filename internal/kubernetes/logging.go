package kubernetes

import (
	log "github.com/sirupsen/logrus"
)

type KLogWriter struct {
	level string
}

func (w *KLogWriter) Write(p []byte) (n int, err error) {
	switch w.level {
	case "FATAL":
		log.WithField("klog", string(p)).Error("kubernetes client-side FATAL error")
	case "ERROR":
		log.WithField("klog", string(p)).Error("kubernetes client-side ERROR error")
	case "WARNING":
		log.WithField("klog", string(p)).Warn("kubernetes client-side WARNING error")
	default:
		log.WithField("klog", string(p)).Info("kubernetes client-side INFO error")
	}

	return len(p), nil
}
