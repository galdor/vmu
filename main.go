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
