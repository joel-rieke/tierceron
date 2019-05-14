GOPATH=/mnt/c/Users/robert.simons/go/:$(shell pwd)/vendor:$(shell pwd)
GOPATH2=/mnt/c/Users/tia.jin/go/:$(shell pwd)/vendor:$(shell pwd)
GOBIN=$(shell pwd)/bin
GOFILES=$(wildcard *.go)

api:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/apiRouter webapi/apiRouter/router.go
config:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/vaultconfig vaultconfig/vaultconfig.go
seed:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/vaultinit vaultinit/init.go
gen:
	protoc --proto_path=. --twirp_out=. --go_out=. rpc/apinator/service.proto

api2:
	@GOPATH2=$(GOPATH2) GOBIN=$(GOBIN) go build -o $(GOBIN)/apiRouter webapi/apiRouter/router.go
config2:
	@GOPATH2=$(GOPATH2) GOBIN=$(GOBIN) go build -o $(GOBIN)/vaultconfig vaultconfig/vaultconfig.go
seed2:
	@GOPATH2=$(GOPATH2) GOBIN=$(GOBIN) go build -o $(GOBIN)/vaultinit vaultinit/init.go