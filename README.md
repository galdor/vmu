# VMU
## Introduction
VMU is a small program to make it easier to deal with virtual machines. It uses
Libvirt and abstract most of the complexity by hard coding as much as possible.

## Usage
Simply install VMU:

```
go install github.com/galdor/vmu
```

Create a profile (a group of VM with common settings running in the same
network):

```
vmu create-profile -f default vm.internal vmu0 192.168.20.0/24
```

Create a VM:

```
vmu create-vm test ./FreeBSD-15.1-RELEASE-amd64-bootonly.iso
```

You can then connect to your VM with e.g. `virt-viewer`.
