module actualat

go 1.24.7

replace golang.org/x/sys => github.com/golang/sys v0.33.0

replace golang.org/x/net => github.com/golang/net v0.40.0

replace golang.org/x/text => github.com/golang/text v0.25.0

replace golang.org/x/crypto => github.com/golang/crypto v0.38.0

require (
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
	github.com/xuri/excelize/v2 v2.9.1
	go.etcd.io/bbolt v1.3.11
)

require (
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/tiendc/go-deepcopy v1.6.0 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.1 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace go.etcd.io/bbolt => github.com/etcd-io/bbolt v1.3.11
