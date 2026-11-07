package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

func CreateStoragePool(client *libvirt.Libvirt, name, dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", dirPath, err)
	}

	absDirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}

	permissions := libvirtxml.StoragePoolTargetPermissions{
		Owner: strconv.Itoa(os.Getuid()),
		Group: strconv.Itoa(os.Getgid()),
		Mode:  "0755",
	}

	target := libvirtxml.StoragePoolTarget{
		Path:        absDirPath,
		Permissions: &permissions,
	}

	poolDefinition := libvirtxml.StoragePool{
		Type:   "dir",
		Name:   name,
		Target: &target,
	}

	poolXMLData, err := poolDefinition.Marshal()
	if err != nil {
		return fmt.Errorf("cannot encode storage pool: %w", err)
	}

	pool, err := client.StoragePoolDefineXML(poolXMLData, 0)
	if err != nil {
		return fmt.Errorf("cannot define storage pool: %w", err)
	}

	if err := client.StoragePoolSetAutostart(pool, 1); err != nil {
		return fmt.Errorf("cannot set storage pool autostart: %w", err)
	}

	active, err := client.StoragePoolIsActive(pool)
	if err != nil {
		return fmt.Errorf("cannot check state of storage pool %w", err)
	}

	if active == 0 {
		if err := client.StoragePoolCreate(pool, 0); err != nil {
			return fmt.Errorf("cannot start storage pool: %w", err)
		}
	}

	return nil
}

func DestroyStoragePoolIfExists(client *libvirt.Libvirt, name string) error {
	pool, err := client.StoragePoolLookupByName(name)
	if err != nil {
		if IsLibvirtErrorCode(err, libvirt.ErrNoStoragePool) {
			return nil
		}

		return fmt.Errorf("cannot lookup storage pool: %w", err)
	}

	active, err := client.StoragePoolIsActive(pool)
	if err != nil {
		return fmt.Errorf("cannot check storage pool state: %w", err)
	}

	if active == 1 {
		err := client.StoragePoolDestroy(pool)
		if err != nil &&
			!IsLibvirtErrorCode(err, libvirt.ErrNoStoragePool) &&
			!IsLibvirtErrorCode(err, libvirt.ErrOperationInvalid) {
			return fmt.Errorf("cannot stop storage pool: %w", err)
		}
	}

	err = client.StoragePoolUndefine(pool)
	if err != nil && !IsLibvirtErrorCode(err, libvirt.ErrNoStoragePool) {
		return fmt.Errorf("cannot undefine storage pool: %w", err)
	}

	return nil
}
