package main

import (
	"encoding/asn1"
)

// ASN1 handling
const (
	TCbeginApp	= 2 // [APPLICATION 2] Begin,
	TCendApp	= 4 // [APPLICATION 4] End,
	TCcontinueApp	= 5 // [APPLICATION 5] Continue,
	TCabortApp	= 7 // [APPLICATION 7] Abort,
)

func decodeTCAP(data []byte) (tag int, otid asn1.RawValue, dtid asn1.RawValue, dialoguePortion asn1.RawValue, components asn1.RawValue, err error) {

	var tmp asn1.RawValue

	// Unpack the choice and ignore it for now
	_, err = asn1.Unmarshal(data, &tmp)
	if err != nil {
		return
	}

	data = tmp.Bytes
	tag = tmp.Tag
	for len(data) > 0 {
		data, err = asn1.Unmarshal(data, &tmp)
		if err != nil {
			return
		}

		switch tmp.Tag {
		case 8:
			otid = tmp
		case 9:
			dtid = tmp
		case 11:
			dialoguePortion = tmp
		case 12:
			components = tmp
		}
	}
	return
}
