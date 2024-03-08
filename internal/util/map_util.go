package util

type Map map[string]string

func (m *Map) MapMerge(source map[string]string, replace bool) {
	if *m == nil {
		*m = make(Map)
	}
	for sourceKey, sourceValue := range source {
		if _, ok := map[string]string(*m)[sourceKey]; !ok || replace {
			map[string]string(*m)[sourceKey] = sourceValue
		}
	}
}
