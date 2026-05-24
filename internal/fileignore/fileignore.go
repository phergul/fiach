package fileignore

var ignoredRegistry = map[string]struct{}{
	".DS_Store": {},
}

func Has(name string) bool {
	_, ignored := ignoredRegistry[name]
	return ignored
}
