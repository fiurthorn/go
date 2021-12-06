package lib

import (
	"fmt"
	"sort"
	"strings"
)

type StringMap map[string]string

func NewStringMap() *StringMap {
	return &StringMap{}
}

func (m *StringMap) Remove(key string) {
	delete(*m, key)
}

func (m *StringMap) Get(key string) (string, bool) {
	elem, ok := (*m)[key]

	return elem, ok
}

func (m *StringMap) Set(key, value string) {
	(*m)[key] = value
}

func (m *StringMap) Keys() []string {
	keys := []string{}
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

func (m *StringMap) Values() []string {
	keys := []string{}
	for _, v := range *m {
		keys = append(keys, v)
	}
	return keys
}

func (m *StringMap) SortedKeys() []string {
	keys := m.Keys()
	sort.Strings(keys)
	return keys
}

func (m *StringMap) String() string {
	values := []string{}
	for k, v := range *m {
		values = append(values, "'"+k+"':'"+v+"'")
	}
	return fmt.Sprintf("StringMap[%s]", strings.Join(values, ", "))
}

func (m StringMap) Len() int {
	return len(m)
}

func (m StringMap) Has(value string) bool {
	_, ok := m[value]
	return ok
}
