package main

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/config"
	gcpCustomAuth "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/k8s/auth/gcp_custom"
	log "github.com/sirupsen/logrus"
)

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	configData, err := config.Load()
	if err != nil {
		log.Fatal(err.Error())
	}

	gcpCustomAuth.RegisterK8SGCPCustomAuthProvider()

	runServer(configData)
}
