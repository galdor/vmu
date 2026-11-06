package main

import "github.com/digitalocean/go-libvirt"

func DomainStateString(state libvirt.DomainState) string {
	var s string

	switch state {
	case libvirt.DomainRunning:
		s = "running"
	case libvirt.DomainBlocked:
		s = "blocked"
	case libvirt.DomainPaused:
		s = "paused"
	case libvirt.DomainShutdown:
		s = "shutdown"
	case libvirt.DomainShutoff:
		s = "shutoff"
	case libvirt.DomainCrashed:
		s = "crashed"
	case libvirt.DomainPmsuspended:
		s = "suspended"
	case libvirt.DomainNostate:
		s = "unknown"
	}

	return s
}
