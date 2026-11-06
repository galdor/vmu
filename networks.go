package main

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"libvirt.org/go/libvirtxml"
)

type Network struct {
	Name           string
	Domain         string
	Bridge         string
	NetworkAddress *ipaddr.IPAddress
}

func CreateNetwork(client *libvirt.Libvirt, network *Network) error {
	data, err := NetworkData(network).Marshal()
	if err != nil {
		return fmt.Errorf("cannot encode network data: %w", err)
	}

	virtNetwork, err := client.NetworkDefineXML(data)
	if err != nil {
		return fmt.Errorf("cannot define network: %w", err)
	}

	if err := client.NetworkSetAutostart(virtNetwork, 1); err != nil {
		return fmt.Errorf("cannot set network autostart: %w", err)
	}

	if err := client.NetworkCreate(virtNetwork); err != nil {
		return err
	}

	return nil
}

func NetworkData(network *Network) *libvirtxml.Network {
	domain := libvirtxml.NetworkDomain{
		Name:      network.Domain,
		LocalOnly: "yes",
	}

	bridge := libvirtxml.NetworkBridge{
		Name:  network.Bridge,
		STP:   "on",
		Delay: "0",
	}

	forward := libvirtxml.NetworkForward{
		Mode: "nat",
	}

	addr := network.NetworkAddress

	netmask := addr.GetNetworkMask()
	firstAddr := addr.GetLower().IncrementSingle().WithoutPrefixLen()
	lastAddr := addr.GetUpper().DecrementSingle().WithoutPrefixLen()

	address := libvirtxml.NetworkIP{
		Address: addr.GetLower().WithoutPrefixLen().String(),
		Netmask: netmask.String(),
		DHCP: &libvirtxml.NetworkDHCP{
			Ranges: []libvirtxml.NetworkDHCPRange{
				{Start: firstAddr.String(), End: lastAddr.String()},
			},
		},
	}

	addresses := []libvirtxml.NetworkIP{address}

	networkData := libvirtxml.Network{
		Name:    network.Name,
		Domain:  &domain,
		Bridge:  &bridge,
		Forward: &forward,
		IPs:     addresses,
	}

	return &networkData
}

func DestroyNetworkIfExists(client *libvirt.Libvirt, name string) error {
	virtNetwork, err := client.NetworkLookupByName(name)
	if err != nil {
		if IsLibvirtErrorCode(err, libvirt.ErrNoNetwork) {
			return nil
		}

		return fmt.Errorf("cannot lookup network: %w", err)
	}

	active, err := client.NetworkIsActive(virtNetwork)
	if err != nil {
		return fmt.Errorf("cannot fetch network state: %w", err)
	}

	if active == 1 {
		if err := client.NetworkDestroy(virtNetwork); err != nil {
			switch {
			case IsLibvirtErrorCode(err, libvirt.ErrNoNetwork):
			case IsLibvirtErrorCode(err, libvirt.ErrOperationInvalid):
			default:
				return fmt.Errorf("cannot stop network: %w", err)
			}
		}
	}

	err = client.NetworkUndefine(virtNetwork)
	if err != nil && !IsLibvirtErrorCode(err, libvirt.ErrNoNetwork) {
		return fmt.Errorf("cannot delete network: %w", err)
	}

	return nil
}
