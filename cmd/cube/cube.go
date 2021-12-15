package main

import (
	"fmt"
	"log"
	"math"
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

type Rotatable interface {
	Rotate(turn quaternion.Quaternion)
	GetAngle() quaternion.Vec3
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

func (t *Ledge) String() string {
	return fmt.Sprintf("{L:%+1.f,%+1.f,%+1.f: %v, %v}", t.Angle.X, t.Angle.Y, t.Angle.Z, t.A, t.B)
}

type Edge struct {
	A, B, C *Tile
	Angle   quaternion.Vec3
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

func NewCubeTile(v quaternion.Vec3) Rotatable {
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

type Cube []Rotatable

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
	for _, t := range *c {
		if t, ok := t.(fmt.Stringer); ok {
			sb.WriteString(t.String())
		} else {
			sb.WriteString("-")
		}
		sb.WriteRune('\n')
	}
	return sb.String()
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

func main() {
	start := time.Now()

	cube := NewCube()
	for i := 0; i < 1419857*17; i++ {
		cube.Rotate(XMinus)
	}
	log.Println(cube)

	log.Println(time.Since(start))

}
