package main

import (
	"uuid"

	"github.com/digitalocean/go-libvirt"

	"go.n16f.net/program"
	"go.n16f.net/vmu/pkg/vmu"
)

func main() {
	p := program.NewProgram("vmu", "VM management utilities")

	p.AddOption("c", "connect", "uri", string(libvirt.QEMUSystem),
		"the URI of the hypervisor to connect to")

	p.AddCommand("list", "list virtual machines", CMDList)

	p.ParseCommandLine()
	p.Run()
}

func CMDList(p *program.Program) {
	client := NewClient(p)
	defer client.Disconnect()

	flags := libvirt.ConnectListDomainsActive |
		libvirt.ConnectListDomainsInactive
	domains, _, err := client.ConnectListAllDomains(1, flags)
	if err != nil {
		p.Fatal("cannot list domains: %v", err)
	}

	t := program.NewTable()
	t.AddColumn(program.TableColumn{Label: "ID"})
	t.AddColumn(program.TableColumn{Label: "NAME"})
	t.AddColumn(program.TableColumn{Label: "UUID"})

	for _, d := range domains {
		t.AddRow(d.ID, d.Name, uuid.UUID(d.UUID))
	}

	t.Print()
}

func NewClient(p *program.Program) *libvirt.Libvirt {
	uriString := p.OptionValue("connect")

	client, err := vmu.NewClient(uriString)
	if err != nil {
		p.Fatal("cannot create client: %v", err)
	}

	return client
}
