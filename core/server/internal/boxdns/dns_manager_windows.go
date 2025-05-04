package boxdns

import (
	"encoding/binary"
	"fmt"
	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing/common/control"
	E "github.com/sagernet/sing/common/exceptions"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"log"
	"nekobox_core/internal/boxdns/winipcfg"
	"net/netip"
	"strings"
)

const (
	nameServerRegistryKey = "NameServer"
	localAddr             = "127.0.0.1"
	dhcpMarkAddr          = "127.1.2.3"
)

var dnsIsSet bool

func (d *DnsManager) HandleSystemDNS(ifc *control.Interface, flag int) {
	if d == nil {
		fmt.Println("No DnsManager, you may need to restart nekoray")
		return
	}
	if !dnsIsSet || ifc == nil {
		return
	}
	if d.lastIfc != nil && d.lastIfc.Index != ifc.Index {
		d.restoreSystemDNS(*d.lastIfc)
	}
	d.setSystemDNS(*ifc)
}

func (d *DnsManager) getInterfaceGuid(ifc control.Interface) (string, error) {
	if d.Monitor == nil {
		return "", E.New("No Dns Manager, you may need to restart nekoray")
	}
	index := ifc.Index
	u, err := ifcIdxtoUUID(index)
	if err != nil {
		return "", err
	}
	guidStr := "{" + u.String() + "}"

	return guidStr, nil
}

func ifcIdxtoUUID(index int) (*uuid.UUID, error) {
	luid, err := winipcfg.LUIDFromIndex(uint32(index))
	if err != nil {
		log.Println("Could not get luid from index")
		return nil, err
	}
	guid, err := luid.GUID()
	if err != nil {
		log.Println("Could not get guid from luid")
		return nil, err
	}
	data1 := make([]byte, 4)
	data2 := make([]byte, 2)
	data3 := make([]byte, 2)
	binary.LittleEndian.PutUint32(data1, guid.Data1)
	binary.LittleEndian.PutUint16(data2, guid.Data2)
	binary.LittleEndian.PutUint16(data3, guid.Data3)
	u, _ := uuid.FromBytes([]byte{
		data1[3], data1[2], data1[1], data1[0],
		data2[1], data2[0],
		data3[1], data3[0],
		guid.Data4[0], guid.Data4[1], guid.Data4[2], guid.Data4[3],
		guid.Data4[4], guid.Data4[5], guid.Data4[6], guid.Data4[7],
	})
	return &u, nil
}

func (d *DnsManager) isIfcDNSDhcp(ifc control.Interface) (dhcp bool, err error) {
	if d == nil {
		fmt.Println("No DnsManager, you may need to restart nekoray")
		return false, E.New("No Dns Manager, you may need to restart nekoray")
	}
	guidStr, err := d.getInterfaceGuid(ifc)
	if err != nil {
		return false, err
	}

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters\\Interfaces\\`+guidStr, registry.QUERY_VALUE)
	if err != nil {
		log.Println("getNameServersForInterface OpenKey:", err)
		return false, err
	}
	defer key.Close()

	if manualNSs, _, err := key.GetStringValue(nameServerRegistryKey); err == nil {
		if len(strings.TrimSpace(manualNSs)) > 0 {
			return false, nil
		}
	}

	return true, nil
}

func (d *DnsManager) restoreSystemDNS(ifx control.Interface) {
	luid, err := winipcfg.LUIDFromIndex(uint32(ifx.Index))
	if err != nil {
		fmt.Println("[restoreSystemDNS] failed to get luid from index:", err)
		return
	}

	dnsServers, err := luid.DNS()
	if err != nil {
		fmt.Println("[restoreSystemDNS] failed to get luid dns servers:", err)
		return
	}

	isDhcp := false
	newDnsServers := make([]netip.Addr, 0)
	for _, server := range dnsServers {
		if server.String() == localAddr || server.String() == dhcpMarkAddr {
			isDhcp = server.String() == dhcpMarkAddr
			continue
		}
		newDnsServers = append(newDnsServers, server)
	}

	if isDhcp {
		err = luid.SetDNS(winipcfg.AddressFamily(windows.AF_INET), nil, nil)
	} else {
		err = luid.SetDNS(winipcfg.AddressFamily(windows.AF_INET), newDnsServers, nil)
	}
	if err != nil {
		fmt.Println("[restoreSystemDNS] failed to set dns servers:", err)
	}

	fmt.Println("[restoreSystemDNS] Local DNS Server removed for:", ifx.Name)
}

func (d *DnsManager) setSystemDNS(ifx control.Interface) {
	luid, err := winipcfg.LUIDFromIndex(uint32(ifx.Index))
	if err != nil {
		fmt.Println("[setSystemDNS] failed to get luid from index:", err)
		return
	}

	dnsServers, err := luid.DNS()
	if err != nil {
		fmt.Println("[setSystemDNS] failed to get luid dns servers:", err)
		return
	}

	hasLocal := false
	newDnsServers := make([]netip.Addr, 0)
	for _, server := range dnsServers {
		if server.String() == localAddr || server.String() == dhcpMarkAddr {
			hasLocal = true
			continue
		}
		newDnsServers = append(newDnsServers, server)
	}
	serverAddr, _ := netip.ParseAddr("127.0.0.1")
	newDnsServers = append([]netip.Addr{serverAddr}, newDnsServers...)
	if !hasLocal {
		dhcp, err := d.isIfcDNSDhcp(ifx)
		if err != nil {
			fmt.Println("[setSystemDNS] failed to determine whether ifc DNS is dhcp:", err)
		}
		if dhcp {
			markAddr, _ := netip.ParseAddr(dhcpMarkAddr)
			newDnsServers = append(newDnsServers, markAddr)
		}
	}

	if err = luid.SetDNS(winipcfg.AddressFamily(windows.AF_INET), newDnsServers, nil); err != nil {
		fmt.Println("[setSystemDNS] failed to set dns servers:", err)
	}

	fmt.Println("[setSystemDNS] Local DNS Server added for:", ifx.Name)
}

func (d *DnsManager) SetSystemDNS(ifc *control.Interface, clear bool) error {
	if d == nil {
		fmt.Println("No DnsManager, you may need to restart nekoray")
		return E.New("No dns Manager, you may need to restart nekoray")
	}

	if ifc == nil {
		ifc = d.Monitor.DefaultInterface()
		if ifc == nil {
			log.Println("Default interface is nil!")
			return E.New("Default interface is nil!")
		}
	}

	if clear {
		dnsIsSet = false
		d.restoreSystemDNS(*ifc)
		return nil
	} else {
		dnsIsSet = true
		d.lastIfc = ifc
		d.setSystemDNS(*ifc)
		return nil
	}
}
