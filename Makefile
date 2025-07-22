GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

VRF_ABI_ARTIFACT := ./abis/DappLinkVRF.sol/DappLinkVRF.json
FACTORY_ABI_ARTIFACT := ./abis/DappLinkVRFFactory.sol/DappLinkVRFFactory.json


dapplink-vrf:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/dapplink-vrf

clean:
	rm dapplink-vrf

test:
	go test -v ./...

lint:
	golangci-lint run ./...

bindings: binding-vrf binding-factory


binding-vrf:
	$(eval temp := $(shell mktemp))

	cat $(VRF_ABI_ARTIFACT) \
    	| jq -r .bytecode.object > $(temp)

	cat $(VRF_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/dapplinkvrf.go \
		--type DappLinkVRF \
		--bin $(temp)

		rm $(temp)

binding-factory:
	$(eval temp := $(shell mktemp))

	cat $(FACTORY_ABI_ARTIFACT) \
    	| jq -r .bytecode.object > $(temp)

	cat $(FACTORY_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/dapplinkfactory.go \
		--type DappLinkVRFFactory \
		--bin $(temp)

		rm $(temp)

.PHONY: \
	dapplink-vrf \
	bindings \
	binding-vrf \
	binding-factory \
	clean \
	test \
	lint