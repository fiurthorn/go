package lib

import (
	"fmt"
	"sort"
	"strings"
)

type void int
type StringSet map[string]void

const empty = void(0)

func NewStringSet() *StringSet {
	return &StringSet{}
}

func (m *StringSet) Remove(key string) {
	delete(*m, key)
}

func (m *StringSet) Add(key, value string) {
	(*m)[key] = empty
}

func (m *StringSet) Values() []string {
	keys := []string{}
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

func (m *StringSet) SortedValues() []string {
	values := m.Values()
	sort.Strings(values)
	return values
}

func (m *StringSet) String() string {
	return fmt.Sprintf("StringSet[%s]", strings.Join(m.Values(), ", "))
}
