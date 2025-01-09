package objects

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestSubsetByPath(t *testing.T) {
	os := ObjectInfoSet{
		"D:/test/01.azw3":    &ObjectInfo{FullPath: "D:/test/01.azw3", File: true},
		"D:/test/02.azw3":    &ObjectInfo{FullPath: "D:/test/02.azw3", File: true},
		"D:/test/01":         &ObjectInfo{FullPath: "D:/test/01", Dir: true},
		"D:/test/01/01.azw3": &ObjectInfo{FullPath: "D:/test/01/01.azw3", File: true},
		"D:/test/01/02.azw3": &ObjectInfo{FullPath: "D:/test/01/02.azw3", File: true},
		"D:/test/02":         &ObjectInfo{FullPath: "D:/test/02", Dir: true},
		"D:/test/02/01.azw3": &ObjectInfo{FullPath: "D:/test/02/01.azw3", File: true},
		"D:/test/02/02.azw3": &ObjectInfo{FullPath: "D:/test/02/02.azw3", File: true},
	}
	// keys := make([]string, 0, len(os))
	// for k := range os {
	// 	keys = append(keys, k)
	// }
	// slices.Sort(keys)
	// for _, k := range keys {
	// 	t.Logf("SET - %s: %s", k, os[k])
	// }
	if len(os) != 8 {
		t.Fatal("Set Size not 8:", len(os))
	}
	t.Log("Set Size:", len(os))

	subset := os.SubsetByPath("D:/test")

	// keys = make([]string, 0, len(subset))
	// for k := range subset {
	// 	keys = append(keys, k)
	// }
	// slices.Sort(keys)
	// for _, k := range keys {
	// 	t.Logf("SUBSET - %s: %s", k, subset[k])
	// }
	if len(subset) != 8 {
		t.Fatal("Subset Size not 8:", len(subset))
	}
	t.Log("Subset Size:", len(subset))

	subset = subset.SubsetByFunc(func(k string, v *ObjectInfo) bool {
		return !v.Dir && slices.Contains([]string{".azw3", ".mobi", ".kfx"}, filepath.Ext(v.FullPath))
	})

	// keys = make([]string, 0, len(subset))
	// for k := range subset {
	// 	keys = append(keys, k)
	// }
	// slices.Sort(keys)
	// for _, k := range keys {
	// 	t.Logf("SUBSET - %s: %s", k, subset[k])
	// }
	if len(subset) != 6 {
		t.Fatal("Subset Size not 6:", len(subset))
	}
	t.Log("SUBSET Size:", len(subset))

	subset = os.SubsetByPath("D:/test/01")

	// keys = make([]string, 0, len(subset))
	// for k := range subset {
	// 	keys = append(keys, k)
	// }
	// slices.Sort(keys)
	// for _, k := range keys {
	// 	t.Logf("SUBSET - %s: %s", k, subset[k])
	// }
	if len(subset) != 2 {
		t.Fatal("Subset Size not 2:", len(subset))
	}
	t.Log("SUBSET Size:", len(subset))
}
