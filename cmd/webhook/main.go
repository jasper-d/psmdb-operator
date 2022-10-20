package main

import (
	"fmt"
	wh "github.com/percona/percona-server-mongodb-operator/pkg/webhook"
	"os"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	GitCommit string
	GitBranch string
	log       = ctrl.Log.WithName("cmd")
)

func printVersion() {
	log.Info(fmt.Sprintf("Git commit: %s Git branch: %s", GitCommit, GitBranch))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	ctrl.SetLogger(zap.New())
	printVersion()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})

	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	hookServer := mgr.GetWebhookServer()
	hookServer.Port = 9001

	hookServer.Register("/host-alias-mutator", &webhook.Admission{Handler: &wh.HostAliasMutator{}})

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited non-zero")
		os.Exit(1)
	}
}
