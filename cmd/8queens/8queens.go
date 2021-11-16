package main

import (
	"log"
	"sort"
	"strings"
)

type Fields [64]rune
type Position struct{ X, Y int }

type Solution []Position

func (x Solution) Len() int           { return len(x) }
func (x Solution) Less(i, j int) bool { return x[i].X < x[j].X }
func (x Solution) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func NewFields() *Fields {
	f := Fields{}
	for i := 0; i < 64; i++ {
		f[i] = ' '
	}
	return &f
}

func (f *Fields) String() string {
	sb := strings.Builder{}

	sb.WriteString("\ny x01234567\n")
	for i := 0; i < n; i++ {
		line := f[n*i : n+n*i]
		sb.WriteString(string(rune('0' + i)))
		sb.WriteString("  ")
		sb.WriteString(string(line))
		sb.WriteRune('\n')
	}

	return sb.String()
}

var (
	n      = 8
	m      = n - 1
	fields = NewFields()

	solution = Solution{}
	uniq     = []Solution{}
	all      = []Solution{}

	counter   int
	solutions int
)

func main() {
	log.Println(fields)
	set(0)
	log.Printf("fields %v %v %v", counter, solutions, fields)
	log.Println(all)
}

func dup(obj Solution) Solution {
	d := make(Solution, len(obj))
	copy(d, obj)

	sort.Sort(d)
	return d
}

func set(y int) {
	if y >= n {
		solutions++
		all = append(all, dup(solution))
		log.Printf("fields %v %v %v", counter, solutions, fields)
		return
	}

	for x := 0; x < n; x++ {
		counter++
		pos := Position{x, y}
		if !underAttack(pos) {
			fields[index(pos)] = 'Q'
			solution = append(solution, Position{x, y})
			set(y + 1)
			solution = solution[:len(solution)-1]
			fields[index(pos)] = ' '
		}
	}
}

func underAttack(pos Position) bool {
	positions := []Position{}
	positions = append(positions, line1(pos)...)
	positions = append(positions, line2(pos)...)
	positions = append(positions, dia1(pos)...)
	positions = append(positions, dia2(pos)...)

	for _, pos := range positions {
		if fields[index(pos)] == 'Q' {
			return true
		}
	}

	return false
}

func draw(fields *Fields, positions []Position) *Fields {
	for _, pos := range positions {
		fields[index(pos)] = 'Q'
	}
	return fields
}

func line1(xy Position) (pos []Position) {
	for i, a, b := 0, 0, xy.Y; a < n; i, a = i+1, a+1 {
		pos = append(pos, Position{a, b})
	}
	return
}

func line2(xy Position) (pos []Position) {
	for i, a, b := 0, xy.X, 0; b < n; i, b = i+1, b+1 {
		pos = append(pos, Position{a, b})
	}
	return
}

func dia1(xy Position) (pos []Position) {
	nc := min(xy.X, xy.Y)
	nd := min(n-xy.X, n-xy.Y)

	for i, a, b := 0, xy.X-nc, xy.Y-nc; a < xy.X+nd; i, a, b = i+1, a+1, b+1 {
		pos = append(pos, Position{a, b})
	}
	return
}

func dia2(xy Position) (pos []Position) {
	na := min(xy.X, m-xy.Y)
	nb := min(m-xy.X, xy.Y)

	for i, a, b := 0, xy.X-na, xy.Y+na; a <= xy.X+nb; i, a, b = i+1, a+1, b-1 {
		pos = append(pos, Position{a, b})
	}
	return
}

func index(pos Position) int {
	if pos.X >= n {
		log.Panicf("x is greater than %d>%d", pos.X, n)
	}
	if pos.Y >= n {
		log.Panicf("y is greater than %d>%d", pos.Y, n)
	}
	return pos.Y*n + pos.X
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
