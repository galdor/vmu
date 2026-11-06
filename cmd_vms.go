package main

import (
	"uuid"

	"github.com/digitalocean/go-libvirt"
	"go.n16f.net/program"
)

func AddVMCommands(p *program.Program) {
	var c *program.Command

	p.AddCommand("create-vm", "create a new VM", cmdCreateVM)

	c = p.AddCommand("destroy-vm", "destroy a VM", cmdDestroyVM)
	c.AddArgument("name", "the name of the VM")

	c = p.AddCommand("start-vm", "start a VM", cmdStartVM)
	c.AddArgument("name", "the name of the VM")

	c = p.AddCommand("stop-vm", "stop a VM", cmdStopVM)
	c.AddArgument("name", "the name of the VM")

	p.AddCommand("list-vms", "list VMs", cmdListVMs)
}

func cmdCreateVM(p *program.Program) {
	// TODO
}

func cmdDestroyVM(p *program.Program) {
	// name := p.ArgumentValue("name")
	// TODO
}

func cmdStartVM(p *program.Program) {
	// name := p.ArgumentValue("name")
	// TODO
}

func cmdStopVM(p *program.Program) {
	// name := p.ArgumentValue("name")
	// TODO
}

func cmdListVMs(p *program.Program) {
	client := ProgramClient()
	defer client.Disconnect()

	flags := libvirt.ConnectListDomainsActive |
		libvirt.ConnectListDomainsInactive
	domains, _, err := client.ConnectListAllDomains(1, flags)
	if err != nil {
		p.Fatal("cannot list domains: %v", err)
	}

	t := program.NewTable()
	t.AddColumn(program.TableColumn{Label: "UUID"})
	t.AddColumn(program.TableColumn{Label: "NAME"})
	t.AddColumn(program.TableColumn{Label: "ID"})
	t.AddColumn(program.TableColumn{Label: "STATE"})

	for _, domain := range domains {
		state, _, err := client.DomainGetState(domain, 0)
		if err != nil {
			p.Fatal("cannot fetch state of domain %q: %v", domain.Name, err)
		}

		t.AddRow(
			uuid.UUID(domain.UUID),
			domain.Name,
			domain.ID,
			DomainStateString(libvirt.DomainState(state)),
		)
	}

	t.Print()
}
