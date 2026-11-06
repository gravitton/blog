package main

import (
	"fmt"

	geom "github.com/gravitton/geometry"
	"github.com/gravitton/grid"
	hex "github.com/gravitton/hexagon"
)

var rows = []string{
	"...............",
	"..#.....#......",
	"..#.....#....G.",
	"..#..~~.#......",
	"....S~~........",
	"..#..~~...##...",
	"..#.......#....",
	"..........#....",
	"...............",
}

func main() {
	shapes()
	hexes()
	grids()
}

func shapes() {
	var walls []geom.Box[float64]
	var scout, goal geom.Point[float64]
	for y, row := range rows {
		for x, tile := range row {
			box := geom.BoxFromMin(geom.Pt(float64(x), float64(y)), geom.SzU(1.0))
			switch tile {
			case '#':
				walls = append(walls, box)
			case 'S':
				scout = box.Center()
			case 'G':
				goal = box.Center()
			}
		}
	}

	body := geom.Circ(scout, 1.0/3)
	motion := goal.Subtract(scout)
	nearest, found := geom.Hit[float64]{}, false
	for _, wall := range walls {
		hit, ok := geom.Sweep[float64](body, motion, wall)
		if ok && (!found || hit.Fraction < nearest.Fraction) {
			nearest, found = hit, true
		}
	}
	fmt.Printf("geometry: the scout swept toward the goal stops %.0f%% of the way, at %.2f, normal %v\n",
		100*nearest.Fraction, nearest.Point, nearest.Normal)

	blocked := geom.CubBez(scout, geom.Pt(6.67, 1.33), geom.Pt(10.67, 1.78), goal)
	clear := geom.CubBez(scout, geom.Pt(9.33, 5.78), geom.Pt(10.67, 3.11), goal)
	fmt.Printf("geometry: the first curve crosses wall edges %d times, the second %d times\n",
		crossings(blocked, walls), crossings(clear, walls))
}

func crossings(curve geom.CubicBezier[float64], walls []geom.Box[float64]) int {
	count := 0
	for _, wall := range walls {
		for edge := range wall.Edges() {
			count += len(curve.IntersectionSegment(edge))
		}
	}

	return count
}

func hexes() {
	ground := map[hex.Hex]bool{}
	walls := map[hex.Hex]bool{}
	var scout, goal hex.Hex
	for y, row := range rows {
		for x, tile := range row {
			h := hex.OffsetOddR.From(geom.Pt(x, y))
			ground[h] = true
			switch tile {
			case '#':
				walls[h] = true
			case 'S':
				scout = h
			case 'G':
				goal = h
			}
		}
	}

	blocked := func(h hex.Hex) bool {
		return walls[h]
	}
	toward := scout.DirectionTo(goal)
	fmt.Printf("hexagon: the scout sees %d hexes of the map within 6, %d of them toward the goal; sees the goal: %v\n",
		onMap(scout.FieldOfView(6, blocked), ground), onMap(scout.FieldOfViewSector(6, toward, 2, blocked), ground),
		scout.HasLineOfSight(goal, blocked))
}

func onMap(hexes []hex.Hex, ground map[hex.Hex]bool) int {
	count := 0
	for _, h := range hexes {
		if ground[h] {
			count++
		}
	}

	return count
}

type Tile struct {
	wall bool
	cost float64
}

func grids() {
	size := geom.Sz(len(rows[0]), len(rows))
	layouts := []struct {
		name string
		grid *grid.Grid[Tile]
	}{
		{"rectangular", grid.NewRect[Tile](size, grid.RectCellScale(17))},
		{"pointy-top", grid.NewHexPointyTop[Tile](size, grid.HexPointyTopCellScale(17))},
	}

	for _, l := range layouts {
		scout, goal := paint(l.grid)
		visible := l.grid.FieldOfView(scout, 6, opaque)
		path := l.grid.Path(scout, goal, walkable, cost)
		fmt.Printf("grid: on the %s layout the scout sees %d cells, its path has %d cells\n",
			l.name, visible.Len(), len(path))
	}
}

func paint(g *grid.Grid[Tile]) (scout, goal grid.Index) {
	g.Fill(Tile{cost: 1})
	for y, row := range rows {
		for x, r := range row {
			index := geom.Pt(x, y)
			switch r {
			case '#':
				g.Cell(index).Set(Tile{wall: true})
			case '~':
				g.Cell(index).Set(Tile{cost: 5})
			case 'S':
				scout = index
			case 'G':
				goal = index
			}
		}
	}

	return scout, goal
}

func opaque(cell grid.Cell[Tile]) bool {
	return cell.Value().wall
}

func walkable(cell grid.Cell[Tile]) bool {
	return !cell.Value().wall
}

func cost(_, next grid.Cell[Tile]) float64 {
	return next.Value().cost
}
