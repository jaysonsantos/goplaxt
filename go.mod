module github.com/xanderstrike/goplaxt

go 1.12

require (
	github.com/DATA-DOG/go-sqlmock v1.3.3
	github.com/alicebob/miniredis/v2 v2.8.0
	github.com/etherlabsio/healthcheck v0.0.0-20191224061800-dd3d2fd8c3f6
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gravitational/trace v1.1.15
	github.com/lib/pq v1.10.3
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.1
)

require (
	github.com/AthenZ/athenz v1.10.50
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/uptrace/opentelemetry-go-extra/otellogrus v0.1.12
	github.com/xanderstrike/plexhooks v0.0.0-20200926011736-c63bcd35fe3e
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.30.0
	go.opentelemetry.io/otel v1.6.3
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.5.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.5.0
	go.opentelemetry.io/otel/metric v0.27.0
	go.opentelemetry.io/otel/sdk v1.5.0
	google.golang.org/grpc v1.45.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/xanderstrike/plexhooks => github.com/jaysonsantos/plexhooks v0.0.0-20220423205150-ba0798c4ca2b
