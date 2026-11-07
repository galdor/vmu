package main

import (
	"fmt"
	"math/big"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func ParseNetworkAddress(s string) (*ipaddr.IPAddress, error) {
	netAddrString := ipaddr.NewIPAddressString(s)

	netAddr, err := netAddrString.ToAddress()
	if err != nil {
		return nil, err
	}

	if !netAddr.IsSinglePrefixBlock() {
		return nil, fmt.Errorf("invalid network address")
	}

	// We always need at least 3 addresses (meaning at least one usable address
	// in the range).
	if netAddr.GetCount().Cmp(big.NewInt(3)) == -1 {
		return nil, fmt.Errorf("network address range too small")
	}

	return netAddr, nil
}

func DHCPIPAddressRange(
	netAddr *ipaddr.IPAddress,
) (*ipaddr.IPAddress, *ipaddr.IPAddress) {
	// Skip the network address and the host address.
	firstAddr := netAddr.GetLower().Increment(2).WithoutPrefixLen()

	// Skip the broadcast address.
	lastAddr := netAddr.GetUpper().DecrementSingle().WithoutPrefixLen()

	return firstAddr, lastAddr
}
