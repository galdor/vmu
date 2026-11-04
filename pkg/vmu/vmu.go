package vmu

import (
	"fmt"
	"net/url"

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
