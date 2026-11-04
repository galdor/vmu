MAKEFLAGS += --no-print-directory

sh = $(strip $(shell $(1))$(if $(filter-out 0,$(.SHELLSTATUS)), \
                               $(error Command failed: $(1))))

git_short_revision := $(call sh,git rev-parse --short HEAD)
build_date := $(call sh,date -u +v%Y%m%dT%H%M%SZ)
build_id ?= $(build_date)-$(git_short_revision)

bin_dir = $(CURDIR)

all: build

build: FORCE
	CGO_ENABLED=0 \
	go build -o $(bin_dir) \
	  -ldflags="-X 'main.BuildId=$(build_id)'" \
	  go.n16f.net/vmu/cmd/vmu

check: vet

vet:
	go vet $(CURDIR)/...

test:
	go test -race -count 1 $(CURDIR)/...

clean:
	$(RM) $(wildcard $(bin_dir)/*)

FORCE:

.PHONY: all build check vet test
