module github.com/cezmunsta/ssh_ms

go 1.16

require (
	github.com/gabriel-vasile/mimetype v1.4.0
	github.com/hashicorp/vault v1.8.5
	github.com/hashicorp/vault-plugin-secrets-kv v0.9.0
	github.com/hashicorp/vault/api v1.1.2-0.20210713235431-1fc8af4c041f
	github.com/hashicorp/vault/sdk v0.2.2-0.20211101151547-6654f4b913f9
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
)

replace github.com/circonus-labs/circonusllhist v0.3.0 => github.com/openhistogram/circonusllhist v0.3.0

replace go.opentelemetry.io/otel/semconv v0.20.0 => github.com/open-telemetry/opentelemetry-go/semconv v1.0.0-RC1

replace google.golang.org/grpc => google.golang.org/grpc v1.29.1
