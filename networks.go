package main

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"net"

	"github.com/digitalocean/go-libvirt"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"libvirt.org/go/libvirtxml"
)

var (
	// Note that changing the prefix size will break GenerateMACAddress.
	MACAddressPrefix     = []byte{0x52, 0x54, 0x01}
	NbUsableMACAddresses = 0xffffff // prefix is 3 bytes so 3 bytes left
)

type Network struct {
	Name           string
	Domain         string
	Bridge         string
	NetworkAddress *ipaddr.IPAddress
}

func CreateNetwork(client *libvirt.Libvirt, network *Network) error {
	xmlData, err := NetworkDefinition(network).Marshal()
	if err != nil {
		return fmt.Errorf("cannot encode network data: %w", err)
	}

	virtNetwork, err := client.NetworkDefineXML(xmlData)
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

func NetworkDefinition(network *Network) *libvirtxml.Network {
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

	netmask := network.NetworkAddress.GetNetworkMask()
	netAddr := network.NetworkAddress.GetLower().IncrementSingle()

	address := libvirtxml.NetworkIP{
		Address: netAddr.WithoutPrefixLen().String(),
		Netmask: netmask.String(),
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

func UpdateNetworkDHCPHost(
	client *libvirt.Libvirt, virtNetwork libvirt.Network,
	cmd libvirt.NetworkUpdateCommand, section libvirt.NetworkUpdateSection,
	host *libvirtxml.NetworkDHCPHost,
) error {
	data, err := host.Marshal()
	if err != nil {
		return fmt.Errorf("cannot encode DHCP host data: %w", err)
	}

	flags := libvirt.NetworkUpdateAffectConfig

	active, err := client.NetworkIsActive(virtNetwork)
	if err != nil {
		return fmt.Errorf("cannot fetch network state: %w", err)
	}

	if active == 1 {
		flags |= libvirt.NetworkUpdateAffectLive
	}

	return client.NetworkUpdate(virtNetwork, uint32(cmd), uint32(section), -1,
		string(data), flags)
}

func CreateDHCPHost(
	client *libvirt.Libvirt, networkName, name string,
	macAddress net.HardwareAddr, ipAddress *ipaddr.IPAddress,
) error {
	virtNetwork, err := client.NetworkLookupByName(networkName)
	if err != nil {
		return fmt.Errorf("cannot lookup network: %w", err)
	}

	err = UpdateNetworkDHCPHost(client, virtNetwork,
		libvirt.NetworkUpdateCommandDelete,
		libvirt.NetworkSectionIPDhcpHost,
		&libvirtxml.NetworkDHCPHost{MAC: macAddress.String()})
	if err != nil && !IsLibvirtErrorCode(err, libvirt.ErrOperationInvalid) {
		return fmt.Errorf("cannot delete DHCP host for MAC address %s: %w",
			macAddress.String(), err)
	}

	host := libvirtxml.NetworkDHCPHost{
		Name: name,
		MAC:  macAddress.String(),
		IP:   ipAddress.WithoutPrefixLen().String(),
	}

	err = UpdateNetworkDHCPHost(client, virtNetwork,
		libvirt.NetworkUpdateCommandAddLast,
		libvirt.NetworkSectionIPDhcpHost, &host)
	if err != nil {
		return err
	}

	return nil
}

func DeleteDHCPHost(client *libvirt.Libvirt, networkName, name string) error {
	virtNetwork, err := client.NetworkLookupByName(networkName)
	if err != nil {
		return fmt.Errorf("cannot lookup network: %w", err)
	}

	err = UpdateNetworkDHCPHost(client, virtNetwork,
		libvirt.NetworkUpdateCommandDelete,
		libvirt.NetworkSectionIPDhcpHost,
		&libvirtxml.NetworkDHCPHost{Name: name})
	if err != nil && !IsLibvirtErrorCode(err, libvirt.ErrOperationInvalid) {
		return err
	}

	return nil
}

func PickMACAddress(client *libvirt.Libvirt) (net.HardwareAddr, error) {
	addrs, err := usedMACAddresses(client)
	if err != nil {
		return nil, err
	}

	start := rand.IntN(NbUsableMACAddresses)

	for i := range NbUsableMACAddresses {
		suffix := (start + i) % NbUsableMACAddresses

		// Depends on the size of the prefix.
		addr := net.HardwareAddr{
			MACAddressPrefix[0], MACAddressPrefix[1], MACAddressPrefix[2],
			byte(suffix >> 16), byte(suffix >> 8), byte(suffix),
		}

		if _, found := addrs[addr.String()]; !found {
			return addr, nil
		}
	}

	return nil, fmt.Errorf("no available MAC address found")
}

func usedMACAddresses(
	client *libvirt.Libvirt,
) (map[string]struct{}, error) {
	domains, _, err := client.ConnectListAllDomains(1, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot list domains: %w", err)
	}

	addrs := make(map[string]struct{})

	for _, domain := range domains {
		flags := []libvirt.DomainXMLFlags{libvirt.DomainXMLInactive}

		active, err := client.DomainIsActive(domain)
		if err != nil {
			if IsLibvirtErrorCode(err, libvirt.ErrNoDomain) {
				continue
			}

			return nil, fmt.Errorf("cannot fetch state of domain %q: %w",
				domain.Name, err)
		}

		if active == 1 {
			flags = append(flags, 0)
		}

		for _, flags := range flags {
			xmlDesc, err := client.DomainGetXMLDesc(domain, flags)
			if err != nil {
				if IsLibvirtErrorCode(err, libvirt.ErrNoDomain) {
					continue
				}

				return nil, fmt.Errorf("cannot fetch XML description of "+
					"domain %q: %w", domain.Name, err)
			}

			var desc libvirtxml.Domain
			if err := desc.Unmarshal(xmlDesc); err != nil {
				return nil, fmt.Errorf("cannot decode XML description of "+
					"domain %q: %w", domain.Name, err)
			}

			if desc.Devices == nil {
				continue
			}

			for _, iface := range desc.Devices.Interfaces {
				if iface.MAC == nil {
					continue
				}

				addr, err := net.ParseMAC(iface.MAC.Address)
				if err != nil || len(addr) != 6 {
					continue
				}

				if bytes.HasPrefix(addr, MACAddressPrefix) {
					addrs[addr.String()] = struct{}{}
				}
			}
		}
	}

	return addrs, nil
}

func PickIPAddress(
	client *libvirt.Libvirt, profile *Profile,
) (*ipaddr.IPAddress, error) {
	networkName := profile.Name

	network, err := client.NetworkLookupByName(networkName)
	if err != nil {
		return nil, fmt.Errorf("cannot lookup network: %w", err)
	}

	networkXMLData, err := client.NetworkGetXMLDesc(network, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch network definition: %w", err)
	}

	var libvirtNetwork libvirtxml.Network
	if err := libvirtNetwork.Unmarshal(networkXMLData); err != nil {
		return nil, fmt.Errorf("cannot decode network: %w", err)
	}

	addrs := make(map[string]struct{})
	for _, addr := range libvirtNetwork.IPs {
		if addr.DHCP != nil {
			for _, host := range addr.DHCP.Hosts {
				addrs[host.IP] = struct{}{}
			}
		}
	}

	firstAddr, lastAddr := DHCPIPAddressRange(profile.NetworkAddress)

	it := profile.NetworkAddress.Iterator()
	for it.HasNext() {
		addr := it.Next()

		if addr.Compare(firstAddr) >= 0 && addr.Compare(lastAddr) <= 0 {
			if _, found := addrs[addr.WithoutPrefixLen().String()]; !found {
				return addr, nil
			}
		}
	}

	return nil, fmt.Errorf("no available IP address found")
}
