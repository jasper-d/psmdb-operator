package main

import (
	"fmt"
	wh "github.com/percona/percona-server-mongodb-operator/pkg/webhook"
	"k8s.io/klog/v2"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	klog.Info(fmt.Sprintf("Git commit: %s Git branch: %s", GitCommit, GitBranch))
	klog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	klog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	klog.InitFlags(nil)
	printVersion()

	cfg, err := config.GetConfig()
	if err != nil {
		klog.Exit(err)
	}

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})

	if err != nil {
		klog.Exit(err)
	}

	hookServer := mgr.GetWebhookServer()
	hookServer.Port = 9001

	hookServer.Register("/host-alias-mutator", &webhook.Admission{Handler: &wh.HostAliasMutator{}})

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Exit(err)
	}
}
