// Much of the knowledge of the KPF/KDF/KFX internals comes from Calibre's KFX
// Conversion Input/Output Plugins created by John Howell <jhowell@acm.org> and
// copyrighted under GPL v3. Visit https://www.mobileread.com/forums for more
// details.
package kfx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
	"unsafe"

	"github.com/amazon-ion/ion-go/ion"
	"github.com/disintegration/imaging"
	"go.uber.org/zap"

	"sync2kindle/thumbs/imgutils"
)

type Reader struct {
	width, height int
	fname         string
	//
	asin, cdetype, assetID, bookID, cover string
	thumbnail                             []byte
}

func (r *Reader) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("asin: %s, cdetype: %s, asset_id: %s, book_id: %s, cover_image: %s", r.asin, r.cdetype, r.assetID, r.bookID, r.cover)
}

func NewReader(fname string, w, h int, log *zap.Logger) (*Reader, error) {
	var r *Reader

	l := log.Named("kfx-reader")
	l.Debug("KFX parse starting",
		zap.String("fname", fname), zap.Int("width", w), zap.Int("height", h))
	defer func(start time.Time) {
		l.Debug("KFX parse finished", zap.Stringer("metadata", r), zap.Duration("elapsed", time.Since(start)))
	}(time.Now())

	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	r = &Reader{fname: fname, width: w, height: h}
	return r, r.extractThumbnail(data)
}

// SaveResult saves extracted thumbnail if any to the requested location.
func (r *Reader) SaveResult(dir string) (string, error) {

	if len(r.thumbnail) == 0 || len(r.asin) == 0 {
		return "", nil
	}

	fileName := "thumbnail_" + r.asin + "_" + r.cdetype + "_portrait.jpg"
	fullName := filepath.Join(dir, fileName)

	if err := os.WriteFile(fullName, r.thumbnail, 0644); err != nil {
		return "", err
	}
	return fileName, nil
}

func (r *Reader) extractThumbnail(data []byte) error {

	contHeaderReader := bytes.NewReader(data)
	contHeader := containerHeader{}
	if err := binary.Read(contHeaderReader, binary.LittleEndian, &contHeader); err != nil {
		return err
	}
	if err := contHeader.validate(); err != nil {
		return err
	}

	contInfo := containerInfo{}
	if err := decodeData(createProlog(), data[contHeader.InfoOffset:contHeader.InfoOffset+contHeader.InfoSize], &contInfo); err != nil {
		return err
	}
	if err := contInfo.validate(); err != nil {
		return err
	}

	if contInfo.DocSymLength == 0 {
		return errors.New("no document symbols found, unsupported KFX type")
	}

	lstProlog := data[contInfo.DocSymOffset : contInfo.DocSymOffset+contInfo.DocSymLength]
	docSymbols, err := decodeST(lstProlog)
	if err != nil {
		return fmt.Errorf("unable to read KFX document symbols: %w", err)
	}

	type entity struct {
		id     uint32
		idType uint32
		data   []byte
	}
	entities := make(map[uint32][]*entity)

	indexTabReader := bytes.NewReader(data[contInfo.IndexTabOffset : contInfo.IndexTabOffset+contInfo.IndexTabLength])
	tabEntry := struct {
		NumID   uint32
		NumType uint32
		Offset  uint64
		Size    uint64
	}{}

	for {
		if err := binary.Read(indexTabReader, binary.LittleEndian, &tabEntry); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("unable to read entity table: %w", err)
		}

		entyStart := tabEntry.Offset + uint64(contHeader.Size)
		if tabEntry.Offset+tabEntry.Size > uint64(len(data)) {
			return fmt.Errorf("entity is out of bounds: %d + %d > %d", tabEntry.Offset, tabEntry.Size, len(data))
		}

		if _, ok := docSymbols.FindByID(uint64(tabEntry.NumID)); !ok {
			return fmt.Errorf("entity ID not found in the symbol table: %d", tabEntry.NumID)
		}
		if _, ok := docSymbols.FindByID(uint64(tabEntry.NumType)); !ok {
			return fmt.Errorf("entity type not found in the symbol table: %d", tabEntry.NumType)
		}

		enty := data[entyStart : entyStart+tabEntry.Size]

		entyHeaderReader := bytes.NewReader(enty)
		entyHeader := entityHeader{}
		if err := binary.Read(entyHeaderReader, binary.LittleEndian, &entyHeader); err != nil {
			return err
		}
		if err := entyHeader.validate(); err != nil {
			return err
		}

		entyInfo := entityInfo{}
		if err := decodeData(lstProlog, enty[len(enty)-entyHeaderReader.Len():entyHeader.Size], &entyInfo); err != nil {
			return err
		}
		if err := entyInfo.validate(); err != nil {
			return err
		}

		entities[tabEntry.NumType] = append(entities[tabEntry.NumType],
			&entity{
				id:     tabEntry.NumID,
				idType: tabEntry.NumType,
				data:   enty[entyHeader.Size:],
			})
	}

	meta, exists := entities[symBookMetadata]
	if !exists {
		// NOTE: KFX input plugin expects case like this and attempts to get to
		// $258 directly, processing it differently I am ignoring this for now
		// as I never seen case like this.
		return errors.New("no book metadata found")
	}
	if len(meta) != 1 {
		return fmt.Errorf("ambiguous metadata, expected single book metadata entity, got %d", len(meta))
	}

	type (
		property struct {
			Key   string `ion:"$492"`
			Value any    `ion:"$307"`
		}
		properties struct {
			Category string     `ion:"$495"`
			Metadata []property `ion:"$258"`
		}
		bookMetadata struct {
			CategorizedMetadata []properties `ion:"$491"`
		}
	)
	var bmd bookMetadata
	if err := decodeData(lstProlog, meta[0].data, &bmd); err != nil {
		return err
	}
	if index := slices.IndexFunc(bmd.CategorizedMetadata, func(p properties) bool {
		if p.Category == "kindle_title_metadata" {
			for _, prop := range p.Metadata {
				switch prop.Key {
				case "ASIN":
					r.asin = prop.Value.(string)
				case "cde_content_type":
					r.cdetype = prop.Value.(string)
				case "asset_id":
					r.assetID = prop.Value.(string)
				case "book_id":
					r.bookID = prop.Value.(string)
				case "cover_image":
					r.cover = prop.Value.(string)
				}
			}
			return true
		}
		return false
	}); index == -1 {
		return errors.New("no kindle book metadata found")
	}

	if r.cdetype != "EBOK" || len(r.asin) == 0 || len(r.cover) == 0 {
		// bail out early - we are only interested in non-personal books
		return nil
	}

	eres, exists := entities[symExternalResource]
	if !exists {
		return errors.New("no external resources found")
	}
	id, ok := docSymbols.FindByName(r.cover)
	if !ok {
		return errors.New("cover image name not found in the symbol table")
	}
	id -= ion.V1SystemSymbolTable.MaxID()

	var index int
	if index = slices.IndexFunc(eres, func(e *entity) bool {
		return uint64(e.id) == id
	}); index == -1 {
		return errors.New("cover image id not found in the external resources")
	}

	c := struct {
		Location string `ion:"$165"`
		// ResourceName any    `ion:"$175"` // ion.SymbolToken
		// Format       any    `ion:"$161"` // ion.SymbolToken
		// Width        int    `ion:"$422"`
		// Height       int    `ion:"$423"`
		// Mime         string `ion:"$162"`
	}{}

	if err := decodeData(lstProlog, eres[index].data, &c); err != nil {
		return err
	}

	imgDataID, ok := docSymbols.FindByName(c.Location)
	if !ok {
		return errors.New("cover image location not found in the symbol table")
	}
	imgDataID -= ion.V1SystemSymbolTable.MaxID()

	media, exists := entities[symRawMedia]
	if !exists {
		return errors.New("no raw media fragments found in a book")
	}
	if index = slices.IndexFunc(media, func(e *entity) bool {
		return uint64(e.id) == imgDataID
	}); index == -1 {
		return errors.New("cover image raw media not found in the external resources")
	}

	var img image.Image
	if img, _, err = image.Decode(bytes.NewReader(media[index].data)); err != nil {
		return fmt.Errorf("unable to decode extracted cover: %w", err)
	}
	imgthumb := imaging.Thumbnail(img, r.width, r.height, imaging.Lanczos)
	if imgthumb == nil {
		return errors.New("unable to resize extracted cover")
	}

	var buf = new(bytes.Buffer)
	if err := imaging.Encode(buf, imgthumb, imaging.JPEG, imaging.JPEGQuality(75)); err != nil {
		return fmt.Errorf("unable to encode produced thumbnail: %w", err)
	}
	buf, _ = imgutils.SetJpegDPI(buf, imgutils.DpiPxPerInch, 300, 300)
	r.thumbnail = buf.Bytes()

	return nil
}

type containerHeader struct {
	Signature  [4]byte
	Version    uint16
	Size       uint32
	InfoOffset uint32
	InfoSize   uint32
}

func (c *containerHeader) validate() error {
	const (
		maxContVersion = 2
	)

	if !bytes.Equal(c.Signature[:], []byte("CONT")) {
		return fmt.Errorf("wrong signature for KFX container: % X", c.Signature[:])
	}
	if c.Version > maxContVersion {
		return fmt.Errorf("unsupported KFX container version: %d", c.Version)
	}
	if uintptr(c.Size) < unsafe.Sizeof(c) {
		return fmt.Errorf("invalid KFX container header size: %d", c.Size)
	}
	return nil
}

type containerInfo struct {
	ContainerId         string `ion:"$409"`
	CompressionType     int    `ion:"$410"`
	DRMScheme           int    `ion:"$411"`
	ChunkSize           int    `ion:"$412"`
	IndexTabOffset      int    `ion:"$413"`
	IndexTabLength      int    `ion:"$414"`
	DocSymOffset        int    `ion:"$415"`
	DocSymLength        int    `ion:"$416"`
	FCapabilitiesOffset int    `ion:"$594"`
	FCapabilitiesLength int    `ion:"$595"`
}

func (c *containerInfo) validate() error {
	const (
		defaultCompressionType = 0
		defaultDRMScheme       = 0
	)

	if c.CompressionType != defaultCompressionType {
		return fmt.Errorf("unsupported KFX container compression type: %d", c.CompressionType)
	}
	if c.DRMScheme != defaultDRMScheme {
		return fmt.Errorf("unsupported KFX container DRM: %d", c.DRMScheme)
	}
	return nil
}

type entityHeader struct {
	Signature [4]byte
	Version   uint16
	Size      uint32
}

func (e *entityHeader) validate() error {
	const (
		maxEntityVersion = 1
	)

	if !bytes.Equal(e.Signature[:], []byte("ENTY")) {
		return fmt.Errorf("wrong signature for KFX entity: % X", e.Signature[:])
	}
	if e.Version > maxEntityVersion {
		return fmt.Errorf("unsupported KFX entity version: %d", e.Version)
	}
	if uintptr(e.Size) < unsafe.Sizeof(e) {
		return fmt.Errorf("invalid KFX entity header size: %d", e.Size)
	}
	return nil
}

type entityInfo struct {
	CompressionType int `ion:"$410"`
	DRMScheme       int `ion:"$411"`
}

func (e *entityInfo) validate() error {
	const (
		defaultCompressionType = 0
		defaultDRMScheme       = 0
	)

	if e.CompressionType != defaultCompressionType {
		return fmt.Errorf("unsupported KFX entity compression type: %d", e.CompressionType)
	}
	if e.DRMScheme != defaultDRMScheme {
		return fmt.Errorf("unsupported KFX entity DRM: %d", e.DRMScheme)
	}
	return nil
}
