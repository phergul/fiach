package fileignore

var IgnoredNames = map[string]struct{}{
	".DS_Store": {},
}

func Has(name string) bool {
	_, ignored := IgnoredNames[name]
	return ignored
}
