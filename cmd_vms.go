package main

import (
	"math"
	"uuid"

	"github.com/digitalocean/go-libvirt"
	"go.n16f.net/program"
)

func AddVMCommands(p *program.Program) {
	var c *program.Command

	c = p.AddCommand("create-vm", "create a new VM", cmdCreateVM)
	c.AddArgument("name", "the name of the VM")
	c.AddArgument("cdrom-image", "the path of the initial CDROM image")
	c.AddOption("", "cpus", "count", "1", "the number of vCPUs")
	c.AddOption("", "memory", "amount", "1",
		"the amount of system memory in gibibytes")
	c.AddOption("", "disk", "size", "10",
		"the size of the main disk in gibibytes")

	c = p.AddCommand("destroy-vm", "destroy a VM", cmdDestroyVM)
	c.AddArgument("name", "the name of the VM")

	p.AddCommand("list-vms", "list VMs", cmdListVMs)
}

func cmdCreateVM(p *program.Program) {
	name := p.ArgumentValue("name")
	cdromImage := p.ArgumentValue("cdrom-image")
	nbCPUs := int(p.IntegerOptionValue2("cpus", 1, math.MaxInt32))
	memory := int(p.IntegerOptionValue2("memory", 1, math.MaxInt32))
	diskSize := int(p.IntegerOptionValue2("disk", 1, math.MaxInt32))

	vm := VM{
		Name:       name,
		Memory:     memory,
		NbCPUs:     nbCPUs,
		DiskSize:   diskSize,
		CDROMImage: cdromImage,
	}

	profile := ProgramProfile()

	client := ProgramClient()
	defer client.Disconnect()

	if err := CreateVM(client, profile, &vm); err != nil {
		p.Fatal("cannot create VM: %v", err)
	}
}

func cmdDestroyVM(p *program.Program) {
	name := p.ArgumentValue("name")

	profile := ProgramProfile()

	client := ProgramClient()
	defer client.Disconnect()

	if err := DestroyVMIfExists(client, profile, name); err != nil {
		p.Fatal("cannot destroy VM: %v", err)
	}
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
