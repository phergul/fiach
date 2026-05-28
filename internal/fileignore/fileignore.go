package fileignore

import "strings"

var ignoredRegistry = map[string]struct{}{
	".DS_Store": {},
	"__MACOSX":  {},
}

func Has(name string) bool {
	// macos junk
	if strings.HasPrefix(name, "._") {
		return true
	}

	_, ignored := ignoredRegistry[name]
	return ignored
}
