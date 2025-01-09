package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	// important  pdb header offsets
	uniqueIDSseed      = 68
	numberOfPdbRecords = 76

	bookLength      = 4
	bookRecordCount = 8
	firstPdbRecord  = 78

	// important rec0 offsets
	lengthOfBook      = 4
	cryptoType        = 12
	mobiHeaderBase    = 16
	mobiHeaderLength  = 20
	mobiType          = 24
	mobiVersion       = 36
	firstNonText      = 80
	titleOffset       = 84
	firstRescRecord   = 108
	firstContentIndex = 192
	lastContentIndex  = 194
	kf8FdstIndex      = 192
	fcisIndex         = 200
	flisIndex         = 208
	srcsIndex         = 224
	srcsCount         = 228
	primaryIndex      = 244
	datpIndex         = 256
	huffOffset        = 112
	huffTableOffset   = 120

	// exth records of interest
	exthASIN          = 113
	exthStartReading  = 116
	exthKF8Offset     = 121
	exthCoverOffset   = 201
	exthThumbOffset   = 202
	exthThumbnailURI  = 129
	exthCDEType       = 501
	exthCDEContentKey = 504
)

func getUInt16(data []byte, ofs int) int {
	return int(binary.BigEndian.Uint16(data[ofs:]))
}

func getInt32(data []byte, ofs int) int {
	// in the up to date mobi format, all those are uint32 but I am yet to encounter a situation when int32 is not enough.
	return int(int32(binary.BigEndian.Uint32(data[ofs:])))
}

func getSectionAddr(data []byte, secno int) (int, int) {

	nsec := getUInt16(data, numberOfPdbRecords)
	if secno < 0 || secno >= nsec {
		panic(fmt.Sprintf("secno %d is out of range [0, %d]", secno, nsec))
	}

	var start, end int
	start = getInt32(data, firstPdbRecord+secno*8)
	if secno == nsec-1 {
		end = len(data)
	} else {
		end = getInt32(data, firstPdbRecord+(secno+1)*8)
	}
	return start, end
}

func getExthParams(rec0 []byte) (int, int, int) {
	ebase := mobiHeaderBase + getInt32(rec0, mobiHeaderLength)
	return ebase, getInt32(rec0, ebase+4), getInt32(rec0, ebase+8)
}

func readExth(rec0 []byte, recnum int) [][]byte {

	var values [][]byte

	ebase, _, enum := getExthParams(rec0)
	ebase += 12

	for enum > 0 {
		exthID := getInt32(rec0, ebase)
		exthLen := getInt32(rec0, ebase+4)
		if exthID == recnum {
			// We might have multiple exths, so build a list.
			values = append(values, rec0[ebase+8:ebase+exthLen])
		}
		enum--
		ebase += exthLen
	}
	return values
}

func readSection(data []byte, secno int) []byte {
	start, end := getSectionAddr(data, secno)
	return data[start:end]
}

// This is specific to go - when encoding jpeg standard encoder does not create JFIF APP0 segment and Kindle does not like it.

// JpegDPIType specifyes type of the DPI units
type JpegDPIType uint8

// DPI units type values
const (
	DpiNoUnits JpegDPIType = iota
	DpiPxPerInch
	DpiPxPerSm
)

var (
	marker = []byte{0xFF, 0xE0}                               // APP0 segment marker
	jfif   = []byte{0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x02} // jfif + version
)

// SetJpegDPI creates JFIF APP0 with provided DPI if segment is missing in image.
func SetJpegDPI(buf *bytes.Buffer, dpit JpegDPIType, xdensity, ydensity int16) (*bytes.Buffer, bool) {

	data := buf.Bytes()

	// If JFIF APP0 segment is there - do not do anything
	if bytes.Equal(data[2:4], marker) {
		return buf, false
	}

	var newbuf = new(bytes.Buffer)

	newbuf.Write(data[:2])
	newbuf.Write(marker)
	binary.Write(newbuf, binary.BigEndian, uint16(0x10)) // length
	newbuf.Write(jfif)
	binary.Write(newbuf, binary.BigEndian, uint8(dpit))
	binary.Write(newbuf, binary.BigEndian, uint16(xdensity))
	binary.Write(newbuf, binary.BigEndian, uint16(ydensity))
	binary.Write(newbuf, binary.BigEndian, uint16(0)) // no thumbnail segment
	newbuf.Write(data[2:])

	return newbuf, true
}
