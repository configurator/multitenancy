package webhook

import (
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	webhooklog = logf.Log.WithName("webhook")
)

// webhookServer represents the webhook server
type webhookServer struct {
	client       client.Client
	scheme       *runtime.Scheme
	deserializer runtime.Decoder
}

// newWebHookServer returns a new webhook server with a bound k8s client
// and a deserializer based on the provided scheme.
func newWebHookServer(client client.Client, scheme *runtime.Scheme) *webhookServer {
	return &webhookServer{
		client:       client,
		scheme:       scheme,
		deserializer: serializer.NewCodecFactory(scheme).UniversalDeserializer(),
	}
}

// newWebhookServerMux returns a new server mux to register to the manager's
// built-in webhook server.
func newWebhookServerMux(client client.Client, scheme *runtime.Scheme) *http.ServeMux {
	mux := http.NewServeMux()
	webhookServer := newWebHookServer(client, scheme)
	mux.HandleFunc("/validate", webhookServer.serve)
	mux.HandleFunc("/version", webhookServer.version)
	return mux
}

// SetupServerHandlers register the validation/version request handlers with the
// provided manager's webhook server.
func SetupServerHandlers(mgr ctrl.Manager, certDir string) {
	server := mgr.GetWebhookServer()
	server.CertDir = certDir
	server.Port = 8443
	mux := newWebhookServerMux(mgr.GetClient(), mgr.GetScheme())
	server.Register("/", mux)
	return
}
