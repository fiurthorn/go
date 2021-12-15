package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/westphae/quaternion"
)

/*
         y    z
         |   /
         |  /
         | /
-x ------+------ x
        /|
       / |
      /  |
    -z  -y
*/

type Tint int

const (
	Black Tint = 1 << iota
	Red
	Green
	Blue
	Cyan
	Magenta
	Yellow
)

func VectorSum(v quaternion.Vec3) float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

func NewColor(v quaternion.Vec3) Tint {
	if VectorSum(v) != 1 {
		return Black
	}
	if v.X > 0 {
		return Red
	} else if v.X < 0 {
		return Green
	} else if v.Y > 0 {
		return Blue
	} else if v.Y < 0 {
		return Cyan
	} else if v.Z > 0 {
		return Magenta
	} else if v.Z < 0 {
		return Yellow
	}
	return Black
}

func (t Tint) String() string {
	switch t {
	case Black:
		return "K"
	case Red:
		return "R"
	case Green:
		return "G"
	case Blue:
		return "B"
	case Cyan:
		return "C"
	case Magenta:
		return "M"
	case Yellow:
		return "Y"
	}
	return " "
}

var (
	XPlus = quaternion.FromEuler(math.Pi/2, 0, 0)
	YPlus = quaternion.FromEuler(0, math.Pi/2, 0)
	ZPlus = quaternion.FromEuler(0, 0, math.Pi/2)

	XMinus = quaternion.FromEuler(-math.Pi/2, 0, 0)
	YMinus = quaternion.FromEuler(0, -math.Pi/2, 0)
	ZMinus = quaternion.FromEuler(0, 0, -math.Pi/2)
)

type CubeTile interface {
	Rotate(turn quaternion.Quaternion)
	GetAngle() quaternion.Vec3
	GetColor(axis quaternion.Quaternion) string
}

type Tile struct {
	Alignment quaternion.Vec3
	Color     Tint
}

func (t *Tile) Rotate(turn quaternion.Quaternion) {
	t.Alignment = t.Alignment.Rotate(turn)
}

func (t *Tile) String() string {
	return fmt.Sprintf("[%+1.f,%+1.f,%+1.f|%s]", t.Alignment.X, t.Alignment.Y, t.Alignment.Z, t.Color)
}

type Flat struct {
	A     *Tile
	Angle quaternion.Vec3
}

func (t *Tile) AxisToDirection(axis quaternion.Quaternion) (x, y, z float64) {
	switch axis {
	case XPlus:
		x = 1
	case XMinus:
		x = -1
	case YPlus:
		y = 1
	case YMinus:
		y = -1
	case ZPlus:
		z = 1
	case ZMinus:
		z = -1
	}
	return
}

func (t *Flat) GetColor(axis quaternion.Quaternion) string {
	x, y, z := t.A.AxisToDirection(axis)
	if Equals(t.A.Alignment.X, x) && Equals(t.A.Alignment.Y, y) && Equals(t.A.Alignment.Z, z) {
		return t.A.Color.String()
	}
	return Black.String()
}

func (t *Flat) String() string {
	return fmt.Sprintf("{F:%+1.f,%+1.f,%+1.f: %v}", t.Angle.X, t.Angle.Y, t.Angle.Z, t.A)
}

func (t *Flat) Rotate(turn quaternion.Quaternion) {
	t.Angle = t.Angle.Rotate(turn)
	t.A.Rotate(turn)
}

func (t *Flat) GetAngle() quaternion.Vec3 {
	return t.Angle
}

type Ledge struct {
	A, B  *Tile
	Angle quaternion.Vec3
}

func (t *Ledge) Rotate(turn quaternion.Quaternion) {
	t.Angle = t.Angle.Rotate(turn)
	t.A.Rotate(turn)
	t.B.Rotate(turn)
}

func (t *Ledge) GetAngle() quaternion.Vec3 {
	return t.Angle
}

func (t *Ledge) GetColor(axis quaternion.Quaternion) string {
	x, y, z := t.A.AxisToDirection(axis)
	if Equals(t.A.Alignment.X, x) && Equals(t.A.Alignment.Y, y) && Equals(t.A.Alignment.Z, z) {
		return t.A.Color.String()
	}
	if Equals(t.B.Alignment.X, x) && Equals(t.B.Alignment.Y, y) && Equals(t.B.Alignment.Z, z) {
		return t.B.Color.String()
	}
	return Black.String()
}

func (t *Ledge) String() string {
	return fmt.Sprintf("{L:%+1.f,%+1.f,%+1.f: %v, %v}", t.Angle.X, t.Angle.Y, t.Angle.Z, t.A, t.B)
}

type Edge struct {
	A, B, C *Tile
	Angle   quaternion.Vec3
}

func (t *Edge) GetColor(axis quaternion.Quaternion) string {
	x, y, z := t.A.AxisToDirection(axis)
	if Equals(t.A.Alignment.X, x) && Equals(t.A.Alignment.Y, y) && Equals(t.A.Alignment.Z, z) {
		return t.A.Color.String()
	}
	if Equals(t.B.Alignment.X, x) && Equals(t.B.Alignment.Y, y) && Equals(t.B.Alignment.Z, z) {
		return t.B.Color.String()
	}
	if Equals(t.C.Alignment.X, x) && Equals(t.C.Alignment.Y, y) && Equals(t.C.Alignment.Z, z) {
		return t.C.Color.String()
	}
	return Black.String()
}

func (t *Edge) String() string {
	return fmt.Sprintf("{E:%+1.f,%+1.f,%+1.f: %v, %v, %v}", t.Angle.X, t.Angle.Y, t.Angle.Z, t.A, t.B, t.C)
}

func (t *Edge) Rotate(turn quaternion.Quaternion) {
	t.Angle = t.Angle.Rotate(turn)
	t.A.Rotate(turn)
	t.B.Rotate(turn)
	t.C.Rotate(turn)
}

func (t *Edge) GetAngle() quaternion.Vec3 {
	return t.Angle
}

func NewVect(x, y, z float64) quaternion.Vec3 {
	return quaternion.Vec3{X: x, Y: y, Z: z}
}

func NewTile(v quaternion.Vec3) *Tile {
	return &Tile{Color: NewColor(v), Alignment: v}
}

func NewFlat(v quaternion.Vec3) *Flat {
	if VectorSum(v) != 1 {
		return nil
	}
	return &Flat{Angle: v, A: NewTile(v)}
}

func NewLedge(v quaternion.Vec3) *Ledge {
	if VectorSum(v) != 2 {
		panic("undefined ledge")
	}
	t := Ledge{Angle: v}
	if v.X == 0 {
		t.A = NewTile(NewVect(0, 0, v.Z))
		t.B = NewTile(NewVect(0, v.Y, 0))
	}
	if v.Y == 0 {
		t.A = NewTile(NewVect(0, 0, v.Z))
		t.B = NewTile(NewVect(v.X, 0, 0))
	}
	if v.Z == 0 {
		t.A = NewTile(NewVect(v.X, 0, 0))
		t.B = NewTile(NewVect(0, v.Y, 0))
	}
	return &t
}

func NewEdge(v quaternion.Vec3) *Edge {
	if VectorSum(v) != 3 {
		panic("undefined edge")
	}
	return &Edge{
		Angle: v,
		A:     NewTile(NewVect(0, 0, v.Z)),
		B:     NewTile(NewVect(0, v.Y, 0)),
		C:     NewTile(NewVect(v.X, 0, 0)),
	}
}

func NewCubeTile(v quaternion.Vec3) CubeTile {
	switch VectorSum(v) {
	case 3:
		return NewEdge(v)
	case 2:
		return NewLedge(v)
	case 1:
		return NewFlat(v)
	}
	return nil
}

type Cube []CubeTile

func NewCube() *Cube {
	var cube = Cube{}
	for z := -1; z < 2; z++ {
		for y := -1; y < 2; y++ {
			for x := -1; x < 2; x++ {
				v := NewVect(float64(x), float64(y), float64(z))
				if int(VectorSum(v)) != 0 {
					cube = append(cube, NewCubeTile(v))
				}
			}
		}
	}
	return &cube
}

func (c *Cube) String() string {
	sb := strings.Builder{}
	sb.WriteRune('\n')
	for i, t := range *c {
		if t, ok := t.(fmt.Stringer); ok {
			sb.WriteString(fmt.Sprintf("%2d: ", i))
			sb.WriteString(t.String())
		} else {
			sb.WriteString("-")
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

func (c *Cube) Draw() string {
	lines := []string{}
	lines = append(lines, "    +---+")
	lines = append(lines, fmt.Sprintf("    |%s%s%s|", c.TilesColor(-1, 1, -1, YPlus), c.TilesColor(0, 1, -1, YPlus), c.TilesColor(1, 1, -1, YPlus)))
	lines = append(lines, fmt.Sprintf("    |%s%s%s|", c.TilesColor(-1, 1, 0, YPlus), c.TilesColor(0, 1, 0, YPlus), c.TilesColor(1, 1, 0, YPlus)))
	lines = append(lines, fmt.Sprintf("    |%s%s%s|", c.TilesColor(-1, 1, 1, YPlus), c.TilesColor(0, 1, 1, YPlus), c.TilesColor(1, 1, 1, YPlus)))
	lines = append(lines, "+---+---+---+---+")
	lines = append(lines, fmt.Sprintf("|%s%s%s|%s%s%s|%s%s%s|%s%s%s|", c.TilesColor(-1, 1, -1, XMinus), c.TilesColor(-1, 1, 0, XMinus), c.TilesColor(-1, 1, 1, XMinus), c.TilesColor(-1, 1, 1, ZPlus), c.TilesColor(0, 1, 1, ZPlus), c.TilesColor(1, 1, 1, ZPlus), c.TilesColor(1, 1, 1, XPlus), c.TilesColor(1, 1, 0, XPlus), c.TilesColor(1, 1, -1, XPlus), c.TilesColor(1, 1, -1, ZMinus), c.TilesColor(0, 1, -1, ZMinus), c.TilesColor(-1, 1, -1, ZMinus)))
	lines = append(lines, fmt.Sprintf("|%s%s%s|%s%s%s|%s%s%s|%s%s%s|", c.TilesColor(-1, 0, -1, XMinus), c.TilesColor(-1, 0, 0, XMinus), c.TilesColor(-1, 0, 1, XMinus), c.TilesColor(-1, 0, 1, ZPlus), c.TilesColor(0, 0, 1, ZPlus), c.TilesColor(1, 0, 1, ZPlus), c.TilesColor(1, 0, 1, XPlus), c.TilesColor(1, 0, 0, XPlus), c.TilesColor(1, 0, -1, XPlus), c.TilesColor(1, 0, -1, ZMinus), c.TilesColor(0, 0, -1, ZMinus), c.TilesColor(-1, 0, -1, ZMinus)))
	lines = append(lines, fmt.Sprintf("|%s%s%s|%s%s%s|%s%s%s|%s%s%s|", c.TilesColor(-1, -1, -1, XMinus), c.TilesColor(-1, -1, 0, XMinus), c.TilesColor(-1, -1, 1, XMinus), c.TilesColor(-1, -1, 1, ZPlus), c.TilesColor(0, -1, 1, ZPlus), c.TilesColor(1, -1, 1, ZPlus), c.TilesColor(1, -1, 1, XPlus), c.TilesColor(1, -1, 0, XPlus), c.TilesColor(1, -1, -1, XPlus), c.TilesColor(1, -1, -1, ZMinus), c.TilesColor(0, -1, -1, ZMinus), c.TilesColor(-1, -1, -1, ZMinus)))
	lines = append(lines, "+---+---+---+---+")
	lines = append(lines, fmt.Sprintf("    |%s%s%s|", c.TilesColor(-1, -1, 1, YMinus), c.TilesColor(0, -1, 1, YMinus), c.TilesColor(1, -1, 1, YMinus)))
	lines = append(lines, fmt.Sprintf("    |%s%s%s|", c.TilesColor(-1, -1, 0, YMinus), c.TilesColor(0, -1, 0, YMinus), c.TilesColor(1, -1, 0, YMinus)))
	lines = append(lines, fmt.Sprintf("    |%s%s%s|", c.TilesColor(-1, -1, -1, YMinus), c.TilesColor(0, -1, -1, YMinus), c.TilesColor(1, -1, -1, YMinus)))
	lines = append(lines, "    +---+")
	return strings.Join(lines, "\n")
}

func (c *Cube) TilesColor(x, y, z float64, axis quaternion.Quaternion) string {
	for _, t := range *c {
		a := t.GetAngle()
		if Equals(a.X, x) && Equals(a.Y, y) && Equals(a.Z, z) {
			return t.GetColor(axis)
		}
	}
	return Black.String()
}

func (c *Cube) Len() int {
	return len(*c)
}

func (c *Cube) Less(i, j int) bool {
	a, b := (*c)[i].GetAngle(), (*c)[j].GetAngle()

	if Less(a.Z, b.Z) ||
		(Equals(a.Z, b.Z) && Less(a.Y, b.Y)) ||
		(Equals(a.Z, b.Z) && Equals(a.Y, b.Y) && Less(a.X, b.X)) {
		return true
	}

	return false

}

func Less(a, b float64) bool {
	return a-b < -0.01
}

func Greater(a, b float64) bool {
	return b-a < -0.01
}

func Equals(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func (c *Cube) Swap(i, j int) {
	l := *c
	l[i], l[j] = l[j], l[i]
}

func (c *Cube) Rotate(rot quaternion.Quaternion) {
	for _, t := range *c {
		t.Rotate(rot)
	}
	sort.Sort(c)
}

func (c *Cube) RotateSlice(slice float64, rot quaternion.Quaternion) {
	var x, y, z bool
	switch rot {
	case XPlus, XMinus:
		x = true
	case YPlus, YMinus:
		y = true
	case ZPlus, ZMinus:
		z = true
	}
	log.Println("rot", x, y, z)
	for _, t := range *c {
		if x && Equals(t.GetAngle().X, slice) ||
			y && Equals(t.GetAngle().Y, slice) ||
			z && Equals(t.GetAngle().Z, slice) {
			t.Rotate(rot)
		}
	}
	sort.Sort(c)
}

func main() {
	start := time.Now()

	cube := NewCube()
	log.Print("\n", cube.Draw())
	rand.Seed(0)
	for i := 0; i < 1000; i++ {
		slice := rand.Intn(2) - 1
		var axis quaternion.Quaternion
		switch rand.Intn(6) {
		case 0:
			axis = XPlus
		case 1:
			axis = XMinus
		case 2:
			axis = YPlus
		case 3:
			axis = YMinus
		case 4:
			axis = ZPlus
		case 5:
			axis = ZMinus
		}
		cube.RotateSlice(float64(slice), axis)
	}
	log.Print("\n", cube.Draw())

	log.Println(time.Since(start))
}
