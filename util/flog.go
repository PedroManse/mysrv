package util

import (
	"time"
	"math"
	"fmt"
)

type color struct {
	fr, fg, fb byte
	br, bg, bb byte
}
type Area struct {
	Name string
	id int
	color color
	enabled bool
}
var areas = []Area{}
var AreaNC Area = Area{"", -1, color{255,255,255, 0, 0, 0}, true}
var AreaDivision string = " "

func GetArea(flag int) Area {
	id := int(math.Logb(float64(flag)))
	if (id<len(areas)) {
		return AreaNC
	}
	return areas[id]
}

func NewArea(name string, colors... byte) int {
	var c color
	if len(colors) == 0 {
		c = color{255, 255, 255, 0, 0, 0}
	} else if len(colors) == 3 {
		c = color{colors[0], colors[1], colors[2], 0, 0, 0}
	} else  if len(colors) == 6 {
		c = color{colors[0], colors[1], colors[2], colors[3], colors[4], colors[5]}
	}

	a := Area{name, 1<<len(areas), c, true}
	areas = append(areas, a)
	return a.id
}

func (A *Area)disable() {
	A.enabled = false
}
func (A *Area)enable() {
	A.enabled = false
}
func (col color)ANSI() string {
	if (col.fr != 0 && col.fb == 0) {
		var ANSI string
		if (col.fr == 0) {
			ANSI+="0"
		}
		//ANSI here
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm", col.fr,col.fg,col.fb, col.br,col.bg,col.bb)
}

func FLog(selectedAreas int, format string, stuff... any) {
	var enabled bool
	var preamble string

	for i:=0;i<len(areas);i++ {
		if ((1<<i) & selectedAreas != 0) {
			area:=areas[i]
			enabled = enabled||area.enabled
			if (preamble != "") {
				preamble+=AreaDivision
			}
			preamble+=area.color.ANSI()+area.Name+AreaNC.color.ANSI()
		}
	}

	if (!enabled) {return}

	var text string
	if (format != "") {
		text = fmt.Sprintf(format, stuff...)
	}
	fmttime := time.Now().Format("01/02 15:04:05")
	fmt.Printf("[%v] %s: %s", fmttime, preamble, text)
}

//TODO: get new Flog system from pedromanse/timecard
var FLOG_ERROR int
func init() {
	FLOG_ERROR = NewArea("ERROR")
}
