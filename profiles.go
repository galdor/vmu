package main

import (
	"fmt"
	"os"
	"path"

	"github.com/digitalocean/go-libvirt"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"go.n16f.net/ark/pkg/ark"
	"go.n16f.net/ark/pkg/ark/yaml"
)

type Profile struct {
	Name                 string            `json:"name"`
	Domain               string            `json:"domain"`
	Bridge               string            `json:"bridge"`
	NetworkAddressString string            `json:"network_address"`
	NetworkAddress       *ipaddr.IPAddress `json:"-"`
}

func (p *Profile) Save() error {
	data, err := yaml.Encode(p)
	if err != nil {
		return fmt.Errorf("cannot encode profile: %w", err)
	}

	filePath := ProfilePath(p.Name)

	dirPath := path.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", dirPath, err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("cannot write %q: %w", filePath, err)
	}

	return nil
}

func (p *Profile) MarshalYAML() ([]byte, error) {
	type Profile2 Profile
	p2 := Profile2(*p)

	p2.NetworkAddressString = p2.NetworkAddress.String()

	return yaml.Encode(&p2)
}

func (p *Profile) UnmarshalYAML(data []byte) error {
	type Profile2 Profile
	p2 := Profile2(*p)

	if err := yaml.Decode(data, &p2); err != nil {
		return err
	}

	var err error
	p2.NetworkAddress, err = ParseNetworkAddress(p2.NetworkAddressString)
	if err != nil {
		return fmt.Errorf("cannot parse network address: %w", err)
	}

	*p = Profile(p2)
	return nil
}

func ProfileDirectoryPath() string {
	homePath, err := os.UserHomeDir()
	if err != nil {
		homePath = "/"
	}

	return path.Join(homePath, ".vmu")
}

func ProfilePath(name string) string {
	return path.Join(ProfileDirectoryPath(), name+".yaml")
}

func LoadProfile(name string) (*Profile, error) {
	profile, err := LoadProfileFile(ProfilePath(name))
	if err != nil {
		return nil, err
	}

	if profile.Name != name {
		return nil, fmt.Errorf("profile name mismatch")
	}

	return profile, nil
}

func LoadProfileFile(filePath string) (*Profile, error) {
	var profile Profile
	if err := ark.LoadCfg(filePath, nil, &profile); err != nil {
		return nil, fmt.Errorf("cannot load profile file: %w", err)
	}

	return &profile, nil
}

func CreateProfile(client *libvirt.Libvirt, profile *Profile) error {
	network := Network{
		Name:           profile.Name,
		Domain:         profile.Domain,
		Bridge:         profile.Bridge,
		NetworkAddress: profile.NetworkAddress,
	}

	if err := CreateNetwork(client, &network); err != nil {
		return fmt.Errorf("cannot create network: %w", err)
	}

	if err := profile.Save(); err != nil {
		return fmt.Errorf("cannot save profile: %w", err)
	}

	return nil
}

func DeleteProfileIfExists(client *libvirt.Libvirt, name string) error {
	if err := DestroyNetworkIfExists(client, name); err != nil {
		return fmt.Errorf("cannot destroy network %q: %w", name, err)
	}

	profilePath := ProfilePath(name)
	if err := os.Remove(profilePath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
