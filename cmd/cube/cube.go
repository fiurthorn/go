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

// +-------------------+
// |         y  -z	   |
// |         |  /	   |
// |         | /       |
// |         |/        |
// |-x ------+------ x |
// |        /|         |
// |       / |         |
// |      /  |         |
// |     z  -y         |
// +-------------------+

/*   / +1
   / 0
 / +1
+--------
|-1
| 0
|+1
+--------
| -1 0 +1
*/

type Tint int
type Rotation quaternion.Quaternion
type Vector quaternion.Vec3

const (
	Black Tint = 1 << iota
	Red
	Green
	Blue
	Cyan
	Magenta
	Yellow
)

func (r Rotation) Rotate(v Vector) Vector {
	return Vector(quaternion.Quaternion(r).RotateVec3(quaternion.Vec3(v)))
}

func (q Rotation) Vector() Vector {
	switch q {
	case XPlus:
		return Vector{X: 1}
	case XMinus:
		return Vector{X: -1}
	case YPlus:
		return Vector{Y: 1}
	case YMinus:
		return Vector{Y: -1}
	case ZPlus:
		return Vector{Z: 1}
	case ZMinus:
		return Vector{Z: -1}
	}
	return Vector{}
}

func (v Vector) Rotate(r Rotation) Vector {
	return r.Rotate(v)
}

func VectorSum(v Vector) float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

func GetColorQ(q Rotation) Tint {
	switch q {
	case XPlus:
		return Red
	case XMinus:
		return Green
	case YPlus:
		return Blue
	case YMinus:
		return Cyan
	case ZPlus:
		return Magenta
	case ZMinus:
		return Yellow
	}
	return Black
}

func GetColor(v Vector) Tint {
	if VectorSum(v) != 1 {
		return Black
	}
	if Equals(v.X, 1) {
		return Red
	} else if Equals(v.X, -1) {
		return Green
	} else if Equals(v.Y, 1) {
		return Blue
	} else if Equals(v.Y, -1) {
		return Cyan
	} else if Equals(v.Z, 1) {
		return Magenta
	} else if Equals(v.Z, -1) {
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
	XPlus = Rotation(quaternion.FromEuler(math.Pi/2, 0, 0))
	YPlus = Rotation(quaternion.FromEuler(0, math.Pi/2, 0))
	ZPlus = Rotation(quaternion.FromEuler(0, 0, math.Pi/2))

	XMinus = Rotation(quaternion.FromEuler(-math.Pi/2, 0, 0))
	YMinus = Rotation(quaternion.FromEuler(0, -math.Pi/2, 0))
	ZMinus = Rotation(quaternion.FromEuler(0, 0, -math.Pi/2))
)

type Turn struct {
	slice int
	axis  Rotation
}

type CubeTile interface {
	Rotate(turn Rotation)
	GetAngle() Vector
	GetColor(axis Rotation) string
	GetSlices() []Turn
}

type Tile struct {
	Alignment Vector
	Color     Tint
}

func (t *Tile) Rotate(turn Rotation) {
	t.Alignment = t.Alignment.Rotate(turn)
}

func (t *Tile) String() string {
	return fmt.Sprintf("[%+1.f,%+1.f,%+1.f|%s]", t.Alignment.X, t.Alignment.Y, t.Alignment.Z, t.Color)
}

func (t *Tile) AxisToDirection(axis Rotation) (x, y, z float64) {
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

type Flat struct {
	A     *Tile
	Angle Vector
}

func (t *Flat) GetSlices() []Turn {
	return []Turn{{0, ZPlus}, {0, ZMinus}, {0, YPlus}, {0, YMinus}, {0, ZPlus}, {0, ZMinus}}
}

func (t *Flat) GetColor(axis Rotation) string {
	x, y, z := t.A.AxisToDirection(axis)
	if Equals(t.A.Alignment.X, x) && Equals(t.A.Alignment.Y, y) && Equals(t.A.Alignment.Z, z) {
		return t.A.Color.String()
	}
	return Black.String()
}

func (t *Flat) String() string {
	return fmt.Sprintf("{F:%+1.f,%+1.f,%+1.f: %v}", t.Angle.X, t.Angle.Y, t.Angle.Z, t.A)
}

func (t *Flat) Rotate(turn Rotation) {
	t.Angle = t.Angle.Rotate(turn)
	t.A.Rotate(turn)
}

func (t *Flat) GetAngle() Vector {
	return t.Angle
}

type Ledge struct {
	A, B  *Tile
	Angle Vector
}

func (t *Ledge) GetSlices() []Turn {
	return []Turn{
		{-1, ZPlus}, {0, ZPlus}, {+1, ZPlus}, {-1, ZMinus}, {0, ZMinus}, {+1, ZMinus},
		{-1, YPlus}, {0, YPlus}, {+1, YPlus}, {-1, YMinus}, {0, YMinus}, {+1, YMinus},
		{-1, ZPlus}, {0, ZPlus}, {+1, ZPlus}, {-1, ZMinus}, {0, ZMinus}, {+1, ZMinus},
	}
}

func (t *Ledge) Rotate(turn Rotation) {
	t.Angle = t.Angle.Rotate(turn)
	t.A.Rotate(turn)
	t.B.Rotate(turn)
}

func (t *Ledge) GetAngle() Vector {
	return t.Angle
}

func (t *Ledge) GetColor(axis Rotation) string {
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
	Angle   Vector
}

func (t *Edge) GetSlices() []Turn {
	return []Turn{
		{-1, ZPlus}, {+1, ZPlus}, {-1, ZMinus}, {+1, ZMinus},
		{-1, YPlus}, {+1, YPlus}, {-1, YMinus}, {+1, YMinus},
		{-1, ZPlus}, {+1, ZPlus}, {-1, ZMinus}, {+1, ZMinus},
	}
}

func (t *Edge) GetColor(axis Rotation) string {
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

func (t *Edge) Rotate(turn Rotation) {
	t.Angle = t.Angle.Rotate(turn)
	t.A.Rotate(turn)
	t.B.Rotate(turn)
	t.C.Rotate(turn)
}

func (t *Edge) GetAngle() Vector {
	return t.Angle
}

func NewVect(x, y, z float64) Vector {
	return Vector{X: x, Y: y, Z: z}
}

func NewTile(v Vector) *Tile {
	return &Tile{Color: GetColor(v), Alignment: v}
}

func NewFlat(v Vector) *Flat {
	if VectorSum(v) != 1 {
		return nil
	}
	return &Flat{Angle: v, A: NewTile(v)}
}

func NewLedge(v Vector) *Ledge {
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

func NewEdge(v Vector) *Edge {
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

func NewCubeTile(v Vector) CubeTile {
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

func (c *Cube) TilesColor(x, y, z float64, axis Rotation) string {
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

func (c *Cube) Rotate(rot Rotation) {
	for _, t := range *c {
		t.Rotate(rot)
	}
	sort.Sort(c)
}

func (c *Cube) Turn(axis, slice int) (s float64, q Rotation) {
	s = float64(slice)
	switch axis {
	case 0:
		q = XPlus
	case 1:
		q = XMinus
	case 2:
		q = YPlus
	case 3:
		q = YMinus
	case 4:
		q = ZPlus
	case 5:
		q = ZMinus
	}
	return
}

func (c *Cube) UTurn(axis, slice int) (s float64, q Rotation) {
	s = float64(slice)
	switch axis {
	case 0:
		q = XMinus
	case 1:
		q = XPlus
	case 2:
		q = YMinus
	case 3:
		q = YPlus
	case 4:
		q = ZMinus
	case 5:
		q = ZPlus
	}
	return
}

func (c *Cube) RotateSlice(slice float64, rot Rotation) {
	var x, y, z bool
	switch rot {
	case XPlus, XMinus:
		x = true
	case YPlus, YMinus:
		y = true
	case ZPlus, ZMinus:
		z = true
	}
	for _, t := range *c {
		if x && Equals(t.GetAngle().X, slice) ||
			y && Equals(t.GetAngle().Y, slice) ||
			z && Equals(t.GetAngle().Z, slice) {
			t.Rotate(rot)
		}
	}
	sort.Sort(c)
}

func (c *Cube) SolveRec(depth int) {

}

func (c *Cube) Solve() {
	col := GetColorQ(YPlus)
	var index int = -1
	for i, t := range *c {
		if t, ok := t.(*Flat); ok {
			log.Println("color", t.A.Color, col)
			if t.A.Color == col {
				index = i
				break
			}
		}
	}
	log.Println("index", index)
}

func main() {
	start := time.Now()

	cube := NewCube()
	log.Print("\n", cube.Draw())
	rand.Seed(0)

	for i := 0; i < 1000; i++ {
		cube.RotateSlice(cube.Turn(rand.Intn(6), rand.Intn(2)-1))
	}
	log.Print("\n", cube.Draw())
	cube.Solve()
	// log.Print("\n", cube)

	log.Println(time.Since(start))
}
