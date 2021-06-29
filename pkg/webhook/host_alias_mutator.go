package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var log = logf.Log.WithName("hostalias-webhook_psmdb")

type HostAliasMutator struct {
	Client client.Client
	decoder *admission.Decoder
}

func (a *HostAliasMutator) Handle(ctx context.Context, req admission.Request) admission.Response{
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil || pod.Annotations["percona.com/dns-zone"] == "" {
		log.Info("No DNS zone annotation found, skipping")
		return admission.Allowed("nop")
	}

	const localhost = "127.0.0.1"
	hostName := fmt.Sprintf("%s.%s", pod.Name, pod.Annotations["percona.com/dns-zone"])
	hostNameAdded := false
	for _, alias := range pod.Spec.HostAliases {
		if alias.IP == localhost {
			log.Info("Appending host name to existing alias", hostName, localhost)
			alias.Hostnames = append(alias.Hostnames, hostName)
			hostNameAdded = true
			break
		}
	}

	if !hostNameAdded {
		log.Info("Appending host alias", hostName, localhost)
		pod.Spec.HostAliases = append(pod.Spec.HostAliases, corev1.HostAlias{IP: localhost, Hostnames: []string{hostName}})
	}

	marshaledPod, err := json.Marshal(pod)

	if err != nil {
		log.Info("Error marshaling pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	log.Info("Returning mutated pod with updated host alias")
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod);
}

func (a *HostAliasMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
