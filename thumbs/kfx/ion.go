package kfx

import (
	"bytes"
	"fmt"

	"github.com/amazon-ion/ion-go/ion"
)

const (
	largestKnownSymbol = 834
	//
	symBookMetadata        = 490
	symCategorizedMetadata = 491
	symExternalResource    = 164
	symThumbnails          = 214
	symRawMedia            = 417
)

var (
	ionBVM            = []byte{0xE0, 1, 0, 0xEA} // binary version marker
	sharedSymbolTable = createSST(largestKnownSymbol)
)

// Actual names for symbols could be obtained by looking at EpubToKFXConverter-4.0.jar from Kindle Previewer 3
// with enum of interest in class file "com.amazon.kaf.c/B.class" presently.
func createSST(maxID uint64) ion.SharedSymbolTable {
	symbols := make([]string, 0, maxID)
	for i := len(ion.V1SystemSymbolTable.Symbols()) + 1; i <= len(ion.V1SystemSymbolTable.Symbols())+int(maxID); i++ {
		symbols = append(symbols, fmt.Sprintf("$%d", i))
	}
	return ion.NewSharedSymbolTable("YJ_symbols", 10, symbols)
}

func createProlog() []byte {
	buf := bytes.Buffer{}
	if err := ion.NewBinaryWriter(&buf, sharedSymbolTable).Finish(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func decodeData(prolog, data []byte, v any) error {
	if err := ion.Unmarshal(append(prolog, data[len(ionBVM):]...), v, sharedSymbolTable); err != nil {
		return err
	}
	return nil
}

func decodeST(data []byte) (ion.SymbolTable, error) {
	r := ion.NewReaderCat(bytes.NewReader(data), ion.NewCatalog(sharedSymbolTable))
	r.Next() // we are not interested in the actual values and in most cases this will return false anyways
	if err := r.Err(); err != nil {
		return nil, err
	}
	return r.SymbolTable(), nil
}
