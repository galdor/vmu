package main

import (
	"path"
	"path/filepath"

	"go.n16f.net/program"
)

func AddProfileCommands(p *program.Program) {
	var c *program.Command

	c = p.AddCommand("create-profile", "create a profile", cmdCreateProfile)
	c.AddArgument("name", "the name of the profile")
	c.AddArgument("domain", "the domain used for all VMs in the profile")
	c.AddArgument("bridge", "the name of the bridge device to create")
	c.AddArgument("network", "the network to use for VM addresses")
	c.AddFlag("f", "force", "delete and recreate the profile and its "+
		"resources if it already exists")

	p.AddCommand("list-profiles", "list profiles", cmdListProfiles)
}

func cmdCreateProfile(p *program.Program) {
	name := p.ArgumentValue("name")
	domain := p.ArgumentValue("domain")
	bridge := p.ArgumentValue("bridge")
	force := p.IsOptionSet("force")

	netAddrString := p.ArgumentValue("network")
	netAddr, err := ParseNetworkAddress(netAddrString)
	if err != nil {
		p.Fatal("cannot parse network address %q: %v", netAddrString, err)
	}

	profile := Profile{
		Name:           name,
		Domain:         domain,
		Bridge:         bridge,
		NetworkAddress: netAddr,
	}

	client := ProgramClient()
	defer client.Disconnect()

	if force {
		if err := DeleteProfileIfExists(client, name); err != nil {
			p.Fatal("cannot delete profile: %v", err)
		}
	}

	if err := CreateProfile(client, &profile); err != nil {
		p.Fatal("cannot create profile: %v", err)
	}
}

func cmdListProfiles(p *program.Program) {
	filePaths, err := filepath.Glob(path.Join(ProfileDirectoryPath(), "*"))
	if err != nil {
		p.Fatal("cannot evaluate glob expression: %v", err)
	}

	t := program.NewTable()
	t.AddColumn(program.TableColumn{Label: "NAME"})
	t.AddColumn(program.TableColumn{Label: "DOMAIN"})
	t.AddColumn(program.TableColumn{Label: "BRIDGE"})
	t.AddColumn(program.TableColumn{Label: "NETWORK"})

	for _, filePath := range filePaths {
		profile, err := LoadProfileFile(filePath)
		if err != nil {
			p.Fatal("cannot load profile from %q: %v", filePath, err)
		}

		t.AddRow(
			profile.Name,
			profile.Domain,
			profile.Bridge,
			profile.NetworkAddress.String(),
		)
	}

	t.Print()
}
