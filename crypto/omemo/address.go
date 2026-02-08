package omemo

import "fmt"

// Address uniquely identifies an OMEMO device.
type Address struct {
	JID      string
	DeviceID uint32
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.JID, a.DeviceID)
}
