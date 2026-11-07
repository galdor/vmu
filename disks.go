package main

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

func CreateDisk(
	client *libvirt.Libvirt, profile *Profile, name string, size int,
) (string, error) {
	destroyResources := true

	defer func() {
		if destroyResources {
			if err := DestroyDiskIfExists(client, profile, name); err != nil {
				p.Error("cannot destroy disk: %v", err)
			}
		}
	}()

	pool, err := client.StoragePoolLookupByName(profile.Name)
	if err != nil {
		return "", fmt.Errorf("cannot lookup storage pool: %w", err)
	}

	volumeName := name + ".qcow2"

	capacity := libvirtxml.StorageVolumeSize{
		Unit:  "GiB",
		Value: uint64(size),
	}

	target := libvirtxml.StorageVolumeTarget{
		Format: &libvirtxml.StorageVolumeTargetFormat{Type: "qcow2"},
	}

	storageVolume := libvirtxml.StorageVolume{
		Name:     volumeName,
		Capacity: &capacity,
		Target:   &target,
	}

	storageVolumeXMLData, err := storageVolume.Marshal()
	if err != nil {
		return "", fmt.Errorf("cannot encode volume: %w", err)
	}

	volume, err := client.StorageVolCreateXML(pool, storageVolumeXMLData, 0)
	if err != nil {
		return "", fmt.Errorf("cannot create volume: %w", err)
	}

	volumePath, err := client.StorageVolGetPath(volume)
	if err != nil {
		return "", fmt.Errorf("cannot fetch volume path: %w", err)
	}

	destroyResources = false

	return volumePath, nil
}

func DestroyDiskIfExists(
	client *libvirt.Libvirt, profile *Profile, name string,
) error {
	pool, err := client.StoragePoolLookupByName(profile.Name)
	if err != nil {
		if IsLibvirtErrorCode(err, libvirt.ErrNoStoragePool) {
			return nil
		}

		return fmt.Errorf("cannot lookup storage pool: %w", err)
	}

	if err := client.StoragePoolRefresh(pool, 0); err != nil {
		return fmt.Errorf("cannot refresh storage pool %w", err)
	}

	volumeName := name + ".qcow2"

	volume, err := client.StorageVolLookupByName(pool, volumeName)
	if err != nil {
		if IsLibvirtErrorCode(err, libvirt.ErrNoStorageVol) {
			return nil
		}

		return fmt.Errorf("cannot lookup volume %q: %w", volumeName, err)
	}

	err = client.StorageVolDelete(volume, 0)
	if err != nil && !IsLibvirtErrorCode(err, libvirt.ErrNoStorageVol) {
		return fmt.Errorf("cannot delete volume %q: %w", volumeName, err)
	}

	return nil
}
