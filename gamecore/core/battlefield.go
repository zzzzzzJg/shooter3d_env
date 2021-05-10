package core

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	"../common"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

type AABB struct {
	Xmin float32
	Xmax float32
	Ymin float32
	Ymax float32
	Zmin float32
	Zmax float32
}

type StaticUnit struct {
	Name string

	Pos      vec3.T
	Scale    vec3.T
	Extent   vec3.T
	Rotation vec4.T
	Vertice  []float32
	Cached   bool
	BB       AABB
}

type BattleField struct {
	CurrentTime  float64
	Map          image.Image
	Bounds       image.Rectangle
	Width        int32
	Height       int32
	Lanes        [2][3][]vec3.T
	Restricted_x float32
	Restricted_y float32
	Restricted_w float32
	Restricted_h float32
	Props        []*StaticUnit
}

func GetCollidePosT(start vec3.T, pos vec3.T, tri []vec3.T) (vec3.T, bool) {
	var collide_pos vec3.T
	P0, P1, P2 := tri[0], tri[1], tri[2]
	D := vec3.Sub(&pos, &start)
	E1 := vec3.Sub(&P1, &P0)
	E2 := vec3.Sub(&P2, &P0)
	S := vec3.Sub(&start, &P0)
	S1 := vec3.Cross(&D, &E2)
	S2 := vec3.Cross(&S, &E1)

	S1E1_dot := vec3.Dot(&S1, &E1)
	S2E2_dot := vec3.Dot(&S2, &E2)
	S1S_dot := vec3.Dot(&S1, &S)
	S2D_dot := vec3.Dot(&S2, &D)

	if S1E1_dot == 0 {
		return collide_pos, false
	}

	t := S2E2_dot / S1E1_dot
	b1 := S1S_dot / S1E1_dot
	b2 := S2D_dot / S1E1_dot
	b0 := 1 - b1 - b2

	if t > 0 && t < 1 && b1 > 0 && b1 < 1 && b2 > 0 && b2 < 1 && b0 > 0 && b0 < 1 {
		collide_pos = *P0.Scale(b0).Add(P1.Scale(b1)).Add(P2.Scale(b2))
		return collide_pos, true
	}

	return collide_pos, false
}

func GetCollidePos(start vec3.T, pos vec3.T, tri []float32) (vec3.T, bool) {
	var collide_pos vec3.T
	var ret bool
	var P0, P1, P2 vec3.T
	P0[0] = tri[0]
	P0[1] = tri[1]
	P0[2] = tri[2]

	P1[0] = tri[5]
	P1[1] = tri[6]
	P1[2] = tri[7]

	P2[0] = tri[10]
	P2[1] = tri[11]
	P2[2] = tri[12]
	collide_pos, ret = GetCollidePosT(start, pos, []vec3.T{P0, P1, P2})
	return collide_pos, ret

}

func (unit *StaticUnit) CheckWithin(pos vec3.T) bool {
	// First, check if pos is within AABB
	if pos[0] > unit.BB.Xmin &&
		pos[0] < unit.BB.Xmax &&
		pos[1] > unit.BB.Ymin &&
		pos[1] < unit.BB.Ymax &&
		pos[2] > unit.BB.Zmin &&
		pos[2] < unit.BB.Zmax {

	} else {
		return false
	}

	// Second, check if pos -> unit-center direction collision with triangle point, and to determin if it's within
	_tri_data_len := 15
	tri_count := len(unit.Vertice) / _tri_data_len

	var if_collide bool

	for _idx := 0; _idx < tri_count; _idx++ {
		_, if_collide = GetCollidePos(pos, unit.Pos, unit.Vertice[_idx*_tri_data_len:(_idx+1)*_tri_data_len])
		if if_collide {
			// inner, outer, definitely outside the box
			return false
		}
	}

	return true
}

func (battle_field *BattleField) Within(pos_x float32, pos_y float32) bool {
	if pos_x > battle_field.Restricted_x &&
		pos_x < battle_field.Restricted_x+battle_field.Restricted_w &&
		pos_y > battle_field.Restricted_y &&
		pos_y < battle_field.Restricted_y+battle_field.Restricted_h {
		return true
	}

	return false
}

type Actor struct {
	Pos      common.Vector3
	Scale    common.Vector3
	Extent   common.Vector3
	Rotation common.Quaternion
}

type Scene struct {
	Actors []Actor
}

func (battle_field *BattleField) LoadProps(filename string) {
	var props []*StaticUnit

	// Json file
	file_handle, err := os.Open(filename)
	if err != nil {
		fmt.Println("Load json battle field static objects failed.", err)
		return
	}

	buffer := make([]byte, 100000)
	read_count, err := file_handle.Read(buffer)
	if err != nil {
		return
	}
	buffer = buffer[:read_count]
	var scene Scene

	if err = json.Unmarshal(buffer, &scene); err == nil {
		for _idx, v := range scene.Actors {
			var prop StaticUnit
			prop.Cached = false
			prop.Name = fmt.Sprintf("prop_%d", _idx)

			prop.BB.Xmax = -math.MaxFloat32
			prop.BB.Xmin = math.MaxFloat32

			prop.BB.Ymax = -math.MaxFloat32
			prop.BB.Ymin = math.MaxFloat32

			prop.BB.Zmax = -math.MaxFloat32
			prop.BB.Zmin = math.MaxFloat32

			prop.Pos[0] = v.Pos.X
			prop.Pos[1] = v.Pos.Y
			prop.Pos[2] = v.Pos.Z

			prop.Scale[0] = v.Scale.X
			prop.Scale[1] = v.Scale.Y
			prop.Scale[2] = v.Scale.Z

			prop.Extent[0] = v.Extent.X
			prop.Extent[1] = v.Extent.Y
			prop.Extent[2] = v.Extent.Z

			prop.Rotation[0] = v.Rotation.W
			prop.Rotation[1] = v.Rotation.X
			prop.Rotation[2] = v.Rotation.Y
			prop.Rotation[3] = v.Rotation.Z

			props = append(props, &prop)
		}

	} else {
		fmt.Println("Error is:", err)
	}

	battle_field.Props = props
}

func (battle_field *BattleField) LoadMap(filename string) []BaseFunc {
	// Init lanes
	// Camp 0
	// Upper lane
	battle_field.Lanes[0][0] = []vec3.T{{280, 20, 0}, {980, 20, 0}, {980, 980, 0}} //[]vec3.T{{16, 32, 0}, {526, 20, 0}, {980, 19, 0}}
	// Middle lane
	battle_field.Lanes[0][1] = []vec3.T{{20, 20, 0}, {408, 511, 0}, {980, 980, 0}}
	// Lower lane
	battle_field.Lanes[0][2] = []vec3.T{{20, 20, 0}, {20, 980, 0}, {980, 980, 0}}

	// Camp 1
	// Upper lane
	battle_field.Lanes[1][0] = []vec3.T{{983, 983, 0}, {980, 20, 0}, {19, 19, 0}}
	// Middle lane
	battle_field.Lanes[1][1] = []vec3.T{{974, 983, 0}, {408, 511, 0}, {19, 19, 0}}
	// Lower lane
	battle_field.Lanes[1][2] = []vec3.T{{974, 983, 0}, {470, 985, 0}, {19, 19, 0}}

	file_handle, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Open file failed:%v", filename)
		return nil
	}

	defer file_handle.Close()
	img, img_type, err := image.Decode(file_handle)
	if err != nil {
		fmt.Printf("Open file failed:%v, type:%v", filename, img_type)
		return nil
	}

	battle_field.Map = img
	/*
		x := 399
		y := 18
		color := img.At(x, y)

		r, g, b, _ := color.RGBA()

		fmt.Printf("color is:%v", color)
	*/

	battle_field.Bounds = img.Bounds()
	battle_field.Width = int32(battle_field.Bounds.Max.X - battle_field.Bounds.Min.X)
	battle_field.Height = int32(battle_field.Bounds.Max.Y - battle_field.Bounds.Min.Y)

	battle_units := []BaseFunc{}
	// Mini clustering
	for idx := int32(0); idx < battle_field.Width; idx += 1 {
		for idy := int32(0); idy < battle_field.Height; idy += 1 {
			color := img.At(int(idx), int(idy))
			r, g, b, _ := color.RGBA()
			var unit BaseFunc
			unit_camp := -1
			unit_id := -1
			switch {
			case r == 0 && g != 0 && b == 0:
				unit_camp = 0
				unit_id = UnitTypeBullet

			case r != 0 && g == 0 && b == 0:
				unit_camp = 1
				unit_id = UnitTypeBullet

			case r == 0 && g != 0 && b != 0:
				unit_camp = 1
				unit_id = UnitTypeAncient

			case r != 0 && g == 0 && b != 0:
				unit_camp = 0
				unit_id = UnitTypeAncient

			case r == 0 && g == 0 && b != 0:
				unit_camp = 2
				unit_id = UnitTypeMonster
			default:
				continue
			}

			has_cluster_core := false
			var pos vec3.T
			pos[0], pos[1] = float32(idx), float32(idy)
			for _, tmp_unit := range battle_units {
				if unit.Camp() != tmp_unit.Camp() {
					continue
				}
				tmp_pos := tmp_unit.Position()

				distance := vec3.Distance(&pos, &tmp_pos)
				if distance < 40 {
					has_cluster_core = true
					break
				}

			}
			if !has_cluster_core {
				unit = HeroMgrInst.Spawn(unit_id, int32(unit_camp), float32(idx), float32(idy))
				battle_units = append(battle_units, unit)
			}
		}
	}
	fmt.Printf("Loaded %v units.", len(battle_units))
	return battle_units
}
