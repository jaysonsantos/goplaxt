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
	github.com/peterbourgon/diskv v0.0.0-20180312054125-0646ccaebea1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.5.1
)

require (
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/xanderstrike/plexhooks v0.0.0-20200926011736-c63bcd35fe3e
	golang.org/x/crypto v0.0.0-20210915214749-c084706c2272 // indirect
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf // indirect
	golang.org/x/sys v0.0.0-20210917161153-d61c044b1678 // indirect
	golang.org/x/term v0.0.0-20210916214954-140adaaadfaf // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/xanderstrike/plexhooks => github.com/jaysonsantos/plexhooks v0.0.0-20200926011736-c63bcd35fe3e
