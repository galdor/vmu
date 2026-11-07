package main

import (
	"errors"
	"fmt"
	"net/url"
	"runtime"

	"github.com/digitalocean/go-libvirt"
)

func NewClient(uriString string) (*libvirt.Libvirt, error) {
	uri, err := url.Parse(uriString)
	if err != nil {
		return nil, fmt.Errorf("cannot parse hypervisor URI: %w", err)
	}

	client, err := libvirt.ConnectToURI(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to hypervisor: %w", err)
	}

	return client, nil
}

func IsLibvirtErrorCode(err error, code libvirt.ErrorNumber) bool {
	var virtErr libvirt.Error
	return errors.As(err, &virtErr) && virtErr.Code == uint32(code)
}

func DefaultHypervisor() string {
	var hypervisor string

	switch runtime.GOOS {
	case "linux":
		hypervisor = "kvm"
	case "freebsd":
		hypervisor = "bhyve"
	default:
		hypervisor = "qemu"
	}

	return hypervisor
}
