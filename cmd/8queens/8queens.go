package main

import (
	"log"
	"sort"
	"strings"
)

type Fields [64]rune

func NewFields() *Fields {
	f := Fields{}
	for i := 0; i < 64; i++ {
		f[i] = ' '
	}
	return &f
}

func (f *Fields) Draw(sol Solution) *Fields {
	for _, s := range sol {
		f[index(s)] = 'Q'
	}
	return f
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

type Position struct{ X, Y int }
type Solution []Position

func (x Solution) Len() int           { return len(x) }
func (x Solution) Less(i, j int) bool { return x[i].X < x[j].X }
func (x Solution) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x Solution) Dup() Solution {
	d := make(Solution, len(x))
	copy(d, x)
	return d
}

func (x Solution) Equals(o Solution) bool {
	if len(x) != len(o) {
		return false
	}

	for i, length := 0, len(x); i < length; i++ {
		if x[i] != o[i] {
			return false
		}
	}

	return true
}

type Solutions struct{ data []Solution }

func (x *Solutions) Store(sol Solution) {
	d := sol.Dup()
	sort.Sort(d)
	x.data = append(x.data, d)
}

type Uniq struct{ data []Solution }

func (x *Uniq) Store(sol Solution) {
	r := x.variations(sol)
	for _, i := range x.data {
		if x.equals(i, r) {
			return
		}
	}
	x.data = append(x.data, r[0])
}

func (x *Uniq) equals(i Solution, r [8]Solution) bool {
	return i.Equals(r[0]) || i.Equals(r[1]) ||
		i.Equals(r[2]) || i.Equals(r[3]) ||
		i.Equals(r[4]) || i.Equals(r[5]) ||
		i.Equals(r[6]) || i.Equals(r[7])
}

func (x *Uniq) variations(sol Solution) (result [8]Solution) {
	result[0] = sol.Dup()
	sort.Sort(result[0])
	result[1] = x.mirror1(sol)
	result[2] = x.mirror2(sol)
	result[3] = x.mirror3(sol)
	result[4] = x.mirror4(sol)
	result[5] = x.rotate(sol)
	result[6] = x.rotate(result[5])
	result[7] = x.rotate(result[6])
	return
}

func (x *Uniq) rotate(sol Solution) Solution {
	d := sol.Dup()
	for i := 0; i < n; i++ {
		c := complex(float32(d[i].X)-q, float32(d[i].Y)-q) * 1i
		d[i].X, d[i].Y = int(real(c)+q), int(imag(c)+q)
	}
	sort.Sort(d)
	return d
}

func (x *Uniq) mirror1(sol Solution) Solution {
	d := sol.Dup()
	for i := 0; i < n; i++ {
		d[i].X = m - d[i].X
	}
	sort.Sort(d)
	return d
}

func (x *Uniq) mirror2(sol Solution) Solution {
	d := sol.Dup()
	for i := 0; i < n; i++ {
		d[i].Y = m - d[i].Y
	}
	sort.Sort(d)
	return d
}

func (x *Uniq) mirror3(sol Solution) Solution {
	d := sol.Dup()
	for i := 0; i < n; i++ {
		d[i].X, d[i].Y = m-d[i].Y, m-d[i].X
	}
	sort.Sort(d)
	return d
}

func (x *Uniq) mirror4(sol Solution) Solution {
	d := sol.Dup()
	for i := 0; i < n; i++ {
		d[i].X, d[i].Y = d[i].Y, d[i].X
	}
	sort.Sort(d)
	return d
}

var (
	n      = 8
	m      = n - 1
	q      = float32(m) / 2
	fields = NewFields()

	solution = Solution{}
	uniq     = Uniq{}
	all      = Solutions{}

	iterationCount int
	solutionCount  int
)

func main() {
	// log.Println(fields)
	set(0)
	for _, u := range uniq.data {
		log.Println(NewFields().Draw(u))
	}
	log.Printf("result %v %v %v %v", iterationCount, solutionCount, len(all.data), len(uniq.data))
}

func set(y int) {
	if y >= n {
		solutionCount++
		all.Store(solution)
		uniq.Store(solution)
		// log.Printf("solution %v %v", iterationCount, solutionCount)
		return
	}

	for x := 0; x < n; x++ {
		iterationCount++
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