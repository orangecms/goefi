package device

import (
	"bytes"
	"encoding/binary"
	"log"

	"github.com/foxboron/goefi/efi/attributes"
	"github.com/foxboron/goefi/efi/util"
)

type EFILoadOption struct {
	Attributes         attributes.Attributes
	FilePathListLength uint16
	Description        []byte
	FilePath           []*EFIDevicePath
	Path               []byte
}

type DevicePathType uint8

const (
	_ DevicePathType = iota
	Hardware
	ACPI
	MessagingDevicePath
	MediaDevicePath
	BIOSBootSpecificationDevicePath
	EndOfHardwareDevicePath DevicePathType = 127
)

type DevicePathSubType uint8

const (
	NewDevicePath   DevicePathSubType = 1
	NoNewDevicePath DevicePathSubType = 255
)

type EFIDevicePath struct {
	Type    DevicePathType
	SubType DevicePathSubType
	Length  [2]uint8
}

type EFIDevicePaths interface {
	Format() string
}

func (e EFIDevicePath) Format() string {
	return "No format"
}

func ParseDevicePath(f *bytes.Reader) []EFIDevicePaths {
	var ret []EFIDevicePaths
	for {
		var efidevice EFIDevicePath
		if err := binary.Read(f, binary.LittleEndian, &efidevice); err != nil {
			log.Fatalf("Failed to parse EFI Device Path: %s", err)
		}
		switch efidevice.Type {
		case Hardware:
			d := ParseHardwareDevicePath(f, &efidevice)
			ret = append(ret, d)
		case ACPI:
			d := ParseACPIDevicePath(f, &efidevice)
			ret = append(ret, d)
		case MediaDevicePath:
			d := ParseMediaDevicePath(f, &efidevice)
			ret = append(ret, d)
		case MessagingDevicePath:
			d := ParseMessagingDevicePath(f, &efidevice)
			ret = append(ret, d)
		case EndOfHardwareDevicePath:
			// log.Printf("Reached end of HardwareDevicePath: %+v\n", efidevice)
			goto end
		default:
			// log.Printf("Not implemented EFIDevicePath type: %+v\n", efidevice)
			goto end
		}
	}
end:
	return ret
}

func ParseEFILoadOption(f *bytes.Reader) *EFILoadOption {
	var bootentry EFILoadOption
	for _, b := range []interface{}{&bootentry.Attributes, &bootentry.FilePathListLength} {
		if err := binary.Read(f, binary.LittleEndian, b); err != nil {
			log.Fatalf("Can't parse EFI Load Option: %s", err)
		}
	}
	bootentry.Description = util.ReadNullString(f)
	return &bootentry
}
