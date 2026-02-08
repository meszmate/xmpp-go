// Package omemo implements XEP-0384 OMEMO Encryption, XEP-0380 EME, and XEP-0454 OMEMO Media Sharing.
package omemo

import (
	"context"
	"encoding/xml"
	"fmt"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/plugins/pubsub"
	"github.com/meszmate/xmpp-go/stanza"
)

const Name = "omemo"

// PEP nodes
const (
	NodeDeviceList = "urn:xmpp:omemo:2:devicelist"
	NodeBundles    = "urn:xmpp:omemo:2:bundles"
)

// Encrypted represents an OMEMO encrypted element.
type Encrypted struct {
	XMLName xml.Name `xml:"urn:xmpp:omemo:2 encrypted"`
	Header  Header   `xml:"header"`
	Payload *Payload `xml:"payload,omitempty"`
}

type Header struct {
	XMLName xml.Name `xml:"header"`
	SID     uint32   `xml:"sid,attr"`
	Keys    []Key    `xml:"keys>key"`
	IV      string   `xml:"iv"`
}

type Key struct {
	XMLName xml.Name `xml:"key"`
	RID     uint32   `xml:"rid,attr"`
	Prekey  bool     `xml:"prekey,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

type Payload struct {
	XMLName xml.Name `xml:"payload"`
	Value   string   `xml:",chardata"`
}

// DeviceList represents the OMEMO device list.
type DeviceList struct {
	XMLName xml.Name `xml:"urn:xmpp:omemo:2 devices"`
	Devices []Device `xml:"device"`
}

type Device struct {
	XMLName xml.Name `xml:"device"`
	ID      uint32   `xml:"id,attr"`
	Label   string   `xml:"label,attr,omitempty"`
}

// Bundle represents an OMEMO key bundle.
type Bundle struct {
	XMLName xml.Name `xml:"urn:xmpp:omemo:2 bundle"`
	SPK     SPK      `xml:"spk"`
	SPKS    string   `xml:"spks"`
	IK      string   `xml:"ik"`
	Prekeys []Prekey `xml:"prekeys>pk"`
}

type SPK struct {
	XMLName xml.Name `xml:"spk"`
	ID      uint32   `xml:"id,attr"`
	Value   string   `xml:",chardata"`
}

type Prekey struct {
	XMLName xml.Name `xml:"pk"`
	ID      uint32   `xml:"id,attr"`
	Value   string   `xml:",chardata"`
}

// EME represents XEP-0380 Explicit Message Encryption.
type EME struct {
	XMLName   xml.Name `xml:"urn:xmpp:eme:0 encryption"`
	Namespace string   `xml:"namespace,attr"`
	Name      string   `xml:"name,attr,omitempty"`
}

type Plugin struct {
	mu       sync.RWMutex
	deviceID uint32
	devices  map[string][]Device // jid -> devices
	params   plugin.InitParams
}

func New(deviceID uint32) *Plugin {
	return &Plugin{
		deviceID: deviceID,
		devices:  make(map[string][]Device),
	}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func (p *Plugin) DeviceID() uint32 { return p.deviceID }

func (p *Plugin) SetDevices(jid string, devices []Device) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.devices[jid] = devices
}

func (p *Plugin) GetDevices(jid string) []Device {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.devices[jid]
}

// NewEME creates an EME element for OMEMO.
func NewEME() EME {
	return EME{
		Namespace: ns.OMEMO,
		Name:      "OMEMO",
	}
}

// PublishDeviceListIQ returns an IQ that publishes the device list to PEP.
func PublishDeviceListIQ(devices ...uint32) *stanza.IQPayload {
	devs := make([]Device, len(devices))
	for i, id := range devices {
		devs[i] = Device{ID: id}
	}
	payload, _ := xml.Marshal(&DeviceList{Devices: devs})
	return &stanza.IQPayload{
		IQ: stanza.IQ{Header: stanza.Header{Type: "set", ID: stanza.GenerateID()}},
		Payload: &pubsub.PubSub{
			Publish: &pubsub.Publish{
				Node:  NodeDeviceList,
				Items: []pubsub.PubItem{{ID: "current", Payload: payload}},
			},
		},
	}
}

// PublishBundleIQ returns an IQ that publishes a key bundle to PEP.
// The bundleXML should be the xml.Marshal output of a Bundle.
func PublishBundleIQ(deviceID uint32, bundleXML []byte) *stanza.IQPayload {
	return &stanza.IQPayload{
		IQ: stanza.IQ{Header: stanza.Header{Type: "set", ID: stanza.GenerateID()}},
		Payload: &pubsub.PubSub{
			Publish: &pubsub.Publish{
				Node:  NodeBundles,
				Items: []pubsub.PubItem{{ID: fmt.Sprintf("%d", deviceID), Payload: bundleXML}},
			},
		},
	}
}

// FetchDeviceListIQ returns an IQ that fetches a user's OMEMO device list.
func FetchDeviceListIQ(target jid.JID) *stanza.IQPayload {
	return &stanza.IQPayload{
		IQ: stanza.IQ{Header: stanza.Header{Type: "get", ID: stanza.GenerateID(), To: target}},
		Payload: &pubsub.PubSub{
			Items: &pubsub.Items{Node: NodeDeviceList},
		},
	}
}

// FetchBundlesIQ returns an IQ that fetches OMEMO bundles for a user.
func FetchBundlesIQ(target jid.JID) *stanza.IQPayload {
	return &stanza.IQPayload{
		IQ: stanza.IQ{Header: stanza.Header{Type: "get", ID: stanza.GenerateID(), To: target}},
		Payload: &pubsub.PubSub{
			Items: &pubsub.Items{Node: NodeBundles},
		},
	}
}

func init() {
	_ = ns.OMEMO
	_ = ns.EME
	_ = ns.OMEMOMedia
}
