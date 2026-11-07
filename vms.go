package main

import (
	"fmt"
	"net"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

type VM struct {
	Name       string
	Memory     int // GiB
	NbCPUs     int
	DiskSize   int // GiB
	CDROMImage string
}

func CreateVM(client *libvirt.Libvirt, profile *Profile, vm *VM) error {
	destroyResources := true

	var domain libvirt.Domain
	haveDomain := false

	networkName := profile.Name
	diskName := vm.Name

	defer func() {
		if destroyResources {
			if haveDomain {
				err := client.DomainUndefineFlags(domain,
					libvirt.DomainUndefineNvram)
				if err != nil {
					p.Error("cannot undefine domain: %v", err)
				}
			}

			if err := DestroyDiskIfExists(client, diskName); err != nil {
				p.Error("cannot destroy disk: %v", err)
			}

			if err := DeleteDHCPHost(client, networkName, vm.Name); err != nil {
				p.Error("cannot delete DHCP host: %v", err)
			}
		}
	}()

	macAddress, err := PickMACAddress(client)
	if err != nil {
		return fmt.Errorf("cannot pick MAC address: %w", err)
	}

	ipAddress, err := PickIPAddress(client, profile)
	if err != nil {
		return fmt.Errorf("cannot pick IP address: %w", err)
	}

	err = CreateDHCPHost(client, networkName, vm.Name, macAddress, ipAddress)
	if err != nil {
		return fmt.Errorf("cannot create DHCP host: %w", err)
	}

	volumePath, err := CreateDisk(client, diskName, vm.DiskSize)
	if err != nil {
		return fmt.Errorf("cannot create disk: %w", err)
	}

	domainDefinition := DomainDefinition(profile, vm, macAddress, volumePath)

	domainData, err := domainDefinition.Marshal()
	if err != nil {
		return fmt.Errorf("cannot encode domain: %w", err)
	}

	domain, err = client.DomainDefineXML(domainData)
	if err != nil {
		return fmt.Errorf("cannot define domain: %w", err)
	}

	haveDomain = true

	if err := client.DomainCreate(domain); err != nil {
		return fmt.Errorf("cannot start domain: %w", err)
	}

	destroyResources = false

	return nil
}

func DomainDefinition(
	profile *Profile, vm *VM, macAddress net.HardwareAddr, volumePath string,
) *libvirtxml.Domain {
	cpu := libvirtxml.DomainCPU{Mode: "host-passthrough"}

	vcpu := libvirtxml.DomainVCPU{
		Value: uint(vm.NbCPUs),
	}

	memory := libvirtxml.DomainMemory{
		Value: uint(vm.Memory),
		Unit:  "GiB",
	}

	osType := libvirtxml.DomainOSType{
		Arch:    "x86_64",
		Machine: "q35",
		Type:    "hvm",
	}

	features := libvirtxml.DomainFeatureList{
		ACPI: &libvirtxml.DomainFeature{},
		APIC: &libvirtxml.DomainFeatureAPIC{},
	}

	os := libvirtxml.DomainOS{
		Type:        &osType,
		BootDevices: []libvirtxml.DomainBootDevice{{Dev: "hd"}, {Dev: "cdrom"}},
	}

	disk := libvirtxml.DomainDisk{
		Device: "disk",
		Driver: &libvirtxml.DomainDiskDriver{Name: "qemu", Type: "qcow2"},
		Source: &libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{File: volumePath},
		},
		Target: &libvirtxml.DomainDiskTarget{Dev: "vda", Bus: "virtio"},
	}

	cdrom := libvirtxml.DomainDisk{
		Device: "cdrom",
		Driver: &libvirtxml.DomainDiskDriver{Name: "qemu", Type: "raw"},
		Source: &libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{File: vm.CDROMImage},
		},
		Target:   &libvirtxml.DomainDiskTarget{Dev: "sda", Bus: "sata"},
		ReadOnly: &libvirtxml.DomainDiskReadOnly{},
	}

	network := libvirtxml.DomainInterfaceSourceNetwork{Network: profile.Name}

	networkInterface := libvirtxml.DomainInterface{
		MAC:    &libvirtxml.DomainInterfaceMAC{Address: macAddress.String()},
		Source: &libvirtxml.DomainInterfaceSource{Network: &network},
		Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio"},
	}

	graphics := libvirtxml.DomainGraphic{
		Spice: &libvirtxml.DomainGraphicSpice{AutoPort: "yes"},
	}

	videoDevice := libvirtxml.DomainVideo{
		Model: libvirtxml.DomainVideoModel{Type: "vga"},
	}

	devices := libvirtxml.DomainDeviceList{
		Disks:      []libvirtxml.DomainDisk{disk, cdrom},
		Interfaces: []libvirtxml.DomainInterface{networkInterface},
		Graphics:   []libvirtxml.DomainGraphic{graphics},
		Videos:     []libvirtxml.DomainVideo{videoDevice},
		Inputs:     []libvirtxml.DomainInput{{Type: "tablet", Bus: "usb"}},
	}

	return &libvirtxml.Domain{
		Type:     profile.Hypervisor,
		Name:     vm.Name,
		CPU:      &cpu,
		VCPU:     &vcpu,
		Memory:   &memory,
		Features: &features,
		OS:       &os,
		Devices:  &devices,
	}
}

func DestroyVMIfExists(
	client *libvirt.Libvirt, profile *Profile, name string,
) error {
	domain, err := client.DomainLookupByName(name)
	if err != nil {
		if !IsLibvirtErrorCode(err, libvirt.ErrNoDomain) {
			return fmt.Errorf("cannot lookup domain: %w", err)
		}
	}

	active, err := client.DomainIsActive(domain)
	if err != nil {
		if IsLibvirtErrorCode(err, libvirt.ErrNoDomain) {
			return nil
		}

		return fmt.Errorf("cannot check domain state: %w", err)
	}

	if active == 1 {
		err := client.DomainDestroy(domain)
		if err != nil &&
			!IsLibvirtErrorCode(err, libvirt.ErrNoDomain) &&
			!IsLibvirtErrorCode(err, libvirt.ErrOperationInvalid) {
			return fmt.Errorf("cannot stop domain: %w", err)
		}
	}

	flags := libvirt.DomainUndefineManagedSave |
		libvirt.DomainUndefineSnapshotsMetadata |
		libvirt.DomainUndefineNvram

	err = client.DomainUndefineFlags(domain, flags)
	if err != nil && !IsLibvirtErrorCode(err, libvirt.ErrNoDomain) {
		return fmt.Errorf("cannot undefine domain %q: %w", domain.Name, err)
	}

	if err := DestroyDiskIfExists(client, name); err != nil {
		return fmt.Errorf("cannot destroy disk: %w", err)
	}

	return nil
}
