package main

import (
	"github.com/digitalocean/go-libvirt"

	"go.n16f.net/program"
)

var (
	p *program.Program
)

func main() {
	p = program.NewProgram("vmu", "VM management utilities")

	p.AddOption("c", "connect", "uri", string(libvirt.QEMUSystem),
		"the URI of the hypervisor to connect to")
	p.AddOption("p", "profile", "name", "default",
		"the name of the profile")

	AddProfileCommands(p)
	AddVMCommands(p)

	p.ParseCommandLine()
	p.Run()
}

func ProgramClient() *libvirt.Libvirt {
	uriString := p.OptionValue("connect")

	client, err := NewClient(uriString)
	if err != nil {
		p.Fatal("cannot create client: %v", err)
	}

	return client
}

func ProgramProfile() *Profile {
	name := p.OptionValue("profile")

	profile, err := LoadProfile(name)
	if err != nil {
		p.Fatal("cannot load profile %q: %v", name, err)
	}

	return profile
}
