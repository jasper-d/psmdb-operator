package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var log = logf.Log.WithName("host-alias-mutator")

type HostAliasMutator struct {
	decoder *admission.Decoder
}

func (a *HostAliasMutator) Handle(ctx context.Context, req admission.Request) admission.Response{
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "Failed to decode request")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil || pod.Annotations["percona.com/dns-zone"] == "" {
		log.Info("percona.com/dns-zone annotation not found, skipping")
		return admission.Allowed("nop")
	}

	const localhost = "127.0.0.1"
	hostName := fmt.Sprintf("%s.%s", pod.Name, pod.Annotations["percona.com/dns-zone"])
	hostNameAdded := false
	for _, alias := range pod.Spec.HostAliases {
		if alias.IP == localhost {
			log.Info("Appending host name to existing alias", "alias", hostName)
			alias.Hostnames = append(alias.Hostnames, hostName)
			hostNameAdded = true
			break
		}
	}

	if !hostNameAdded {
		log.Info("Appending host alias", "alias", hostName)
		pod.Spec.HostAliases = append(pod.Spec.HostAliases, corev1.HostAlias{IP: localhost, Hostnames: []string{hostName}})
	}

	podJson, err := json.Marshal(pod)

	if err != nil {
		log.Error(err, "Error marshaling pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	log.Info("Returning mutated pod with updated host alias")
	return admission.PatchResponseFromRaw(req.Object.Raw, podJson)
}

func (a *HostAliasMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
