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

const dnsAnnotationKey = "percona.com/dns-zone"
const loopback = "127.0.0.1"

var log = logf.Log.WithName("host-alias-mutator")

type HostAliasMutator struct {
	decoder *admission.Decoder
}

func (a *HostAliasMutator) Handle(_ context.Context, req admission.Request) admission.Response{
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)

	if err != nil {
		log.Error(err, "Failed to decode request")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil || pod.Annotations[dnsAnnotationKey] == "" {
		log.Info(fmt.Sprintf("%s annotation not found, skipping", dnsAnnotationKey))
		return admission.Allowed("nop")
	}

	externalFqdn := fmt.Sprintf("%s.%s", pod.Name, pod.Annotations[dnsAnnotationKey])

	externalFqdnAdded := false
	for i, alias := range pod.Spec.HostAliases {
		if alias.IP == loopback {
			log.Info("Extending existing host alias", "alias", externalFqdn)
			pod.Spec.HostAliases[i].Hostnames = append(alias.Hostnames, externalFqdn)
			externalFqdnAdded = true
			break
		}
	}

	if !externalFqdnAdded{
		log.Info("Appending new host alias", "alias", externalFqdn)
		pod.Spec.HostAliases = append(pod.Spec.HostAliases, corev1.HostAlias{IP: loopback, Hostnames: []string{externalFqdn}})
	}

	podJson, err := json.Marshal(pod)

	if err != nil {
		log.Error(err, "Error marshaling pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, podJson)
}

func (a *HostAliasMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
