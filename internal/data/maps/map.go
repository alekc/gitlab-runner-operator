package maps

func InitIfNil(mapVar *map[string]string) {
	if *mapVar == nil {
		*mapVar = make(map[string]string)
	}
}

func InitSliceIfNil(mapVar *map[string][]string) {
	if *mapVar == nil {
		*mapVar = make(map[string][]string)
	}
}
