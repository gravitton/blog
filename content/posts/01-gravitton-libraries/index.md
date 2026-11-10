---
title: "Go libraries for 2D games: geometry, hexagon and grid"
date: 2026-11-06
slug: gravitton-libraries
draft: false
description: "Three open-source Go libraries for 2D games, geometry, hexagon and grid, take one task on one map: what a unit sees and how it reaches its goal, each at its own level."
keywords: ["Go", "2D game library", "geometry", "hexagonal grid", "grid", "pathfinding", "field of view", "collision detection", "game engine"]
---

A 2D game asks the same two things of its map over and over: what can a unit see, and how does it get where it is
going. [Gravitton](https://github.com/gravitton) is a set of Go libraries for 2D games, written while building a
game engine on top of them, and three of them take that task, each at its own level:

- [`geometry`](https://github.com/gravitton/geometry): points, vectors, shapes, paths and the collisions between
  them.
- [`hexagon`](https://github.com/gravitton/hexagon): hexagonal coordinates, ranges, lines and sight, with no pixels
  and no storage.
- [`grid`](https://github.com/gravitton/grid): a value per cell on a rectangular, isometric or hexagonal layout,
  with sight and pathfinding, built on the other two.

Beside them are smaller ones: [`spatial`](https://github.com/gravitton/spatial) for the broad phase of collision,
[`container`](https://github.com/gravitton/container) for the storage under a grid,
[`errors`](https://github.com/gravitton/errors) and [`assert`](https://github.com/gravitton/assert). This post is
the map of the series: what the three libraries do on one map, the rules they share, and the posts that each take
one decision from them and show the code and the measurements behind it.

## The task

The README of each of the three opens with the same map:

```go
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
```

`#` is a wall, `~` is rough ground, `S` is a scout and `G` its goal. What does the scout see, and how does it get to
the goal? Each library answers in its own terms, and the picture at the top of its README draws the answer.

## geometry

`geometry` takes the task in continuous space, where a wall is a shape and the scout a body that moves. Its shapes
are the circle, capsule, sector, polygon, rectangle, regular polygon, box and ellipse; its paths the segment, ray,
line, polyline, arc and Bézier curves. Between them it answers intersections, containment, raycasts, nearest points,
contacts and sweeps, and a `Shape` interface holds any shape without knowing which one. It knows nothing about
tiles.

![The map drawn with geometry: the scout's view cast by rays that stop at the walls, with one ray reflected until it reaches the goal; the scout swept toward the goal until it hits a wall; and a curved route to the goal, a first try crossing a wall, the second clear of the walls](shapes.svg "Raycast, Sweep and IntersectionSegment on the map, drawn from what each call returns.")
{.plain}

In its hero a wall is a box and the scout a circle with a radius of a third of a tile. A sweep moves the circle along a
motion and reports where it first touches a shape:

```go
body := geom.Circ(scout, 1.0/3)
hit, ok := geom.Sweep[float64](body, goal.Subtract(scout), wall)
```

The hit holds the contact point, the normal there and the fraction of the motion made. Swept toward the goal, the
scout gets 35% of the way before the nearest wall stops it.

Every type is generic over its number type, so a `Point[int]` is a tile and a `Point[float64]` a position, with the
same methods written once. Every value is immutable: a method returns a new value and never changes its receiver.
It works in screen space, the origin top-left and `+Y` down, and comes with direction enums, `image` interop, JSON,
parsing and `geomtest` assertions.

## hexagon

`hexagon` takes it on hexes, as the math of a hexagonal grid. A `hex.Hex` is two integers in axial coordinates and a
map key. It knows its neighbors, its distances, its rings, spirals and sectors, the hexes on a line and what it can
see; it rotates and reflects in cube space, and converts losslessly to and from seven coordinate systems: axial,
offset and doubled. It knows nothing about where a hex is drawn or what a game stores in it.

![The map in hexes: the scout's field of view, its line to the goal, hidden behind a wall, and its field of view in a sector toward the goal](hexes.svg "FieldOfView, HasLineOfSight and FieldOfViewSector on the map, drawn from what each call returns.")
{.plain}

The game says what blocks sight, and the library asks it:

```go
blocked := func(h hex.Hex) bool {
	return walls[h]
}

visible := scout.FieldOfView(6, blocked)
seen := scout.HasLineOfSight(goal, blocked)
```

On the hero map the scout sees 66 hexes within six, its own included, and the goal is hidden behind a wall. Line
of sight is exact and computed in integers, so it is symmetric: if the scout sees a hex, that hex sees the scout.

Every traversal has a `Seq` form, an iterator that walks the same hexes without allocating a slice for them.

## grid

`grid` takes the whole task, because it is where a map lives. A `Grid[T]` stores one value of any type per cell, in
one flat array, on a rectangular, isometric, flat-top or pointy-top hexagonal layout behind one API. It finds the
cell under a point and the center, bounds and polygon of every cell; it gives neighbors, ranges, rings and lines,
line of sight and field of view; it searches with A*, Dijkstra, greedy best-first and breadth-first; and it builds
regions of cells with set operations, borders, growth and components. It builds on `geometry` for its points and
shapes and on `hexagon` for its hexagonal coordinates.

![The map on a rectangular, an isometric and a pointy-top grid: on each the scout's field of view and its cheapest path to the goal around the rough ground](grids.svg "FieldOfView and Path on three layouts of the map, drawn from what each call returns.")
{.plain}

Here each cell holds a `Tile` with a wall flag and a cost, and the same code runs on every layout; only the
constructor changes:

```go
g := grid.NewRect[Tile](size, grid.RectCellScale(17))

visible := g.FieldOfView(scout, 6, opaque)
path := g.Path(scout, goal, walkable, cost)
```

`opaque`, `walkable` and `cost` are callbacks that read the tile, so the grid never learns what a wall is.
`RectCellScale(17)` is the width of a cell when it is drawn; none of the answers depend on it. On the rectangular
layout the scout sees 61 cells and its path to the goal has 16 cells, both ends included. On the pointy-top layout
it sees 66, the same hexes `hexagon` gave above, because a hexagonal grid uses its sight unchanged.

## What they share

The three libraries share their rules. `geometry` sets them, and the other two follow them where they apply. Each
rule has a post of its own in this series, with the measurements that led to it:

- **Generic over number types**: one set of `geometry` methods for every integer and float type; `hexagon` and
  `grid` index their cells with integers. What that costs, and the compiler flag that takes it back, is
  {{< post "generic-numbers" "what generics over number types cost in Go" >}}.
- **Closed boundaries within a tolerance**: in `geometry`, a point on the edge of a shape is inside it, and two
  shapes that touch collide. Why, and where the rule is still wrong, is
  {{< post "closed-boundaries" "closed boundaries within a tolerance" >}}.
- **The same bits on amd64 and arm64**: in `geometry`, trigonometry aside, and in the fractional hexes of
  `hexagon`, so a replay recorded on one machine plays the same on another. The instruction that breaks it is
  {{< post "fused-multiply-add" "fused multiply-add" >}}.
- **No hidden allocations**: a slice result allocates at its final size, the documentation states every allocation
  beyond that, and a `Seq` form allocates nothing.
- **Degenerate inputs answer**: a zero-length segment or an empty polygon gives an answer, never a panic. A panic is
  only for an argument with no meaning, and it is an error that `errors.Is` matches, as
  {{< post "errors" "the post on structured errors" >}} shows.

Each library is tested to full coverage, with property checks and fuzz targets, on amd64 and arm64; how is
{{< post "testing" "a post of its own" >}}.

## The series

After the rules come the parts a game calls every frame, one post each:

- {{< post "collisions" "2D collision detection" >}}: contacts, sweeps and raycasts in `geometry`, and why a
  bullet never tunnels through a wall.
- {{< post "spatial-hash" "A spatial hash for the broad phase" >}}: which pairs are worth testing, measured against
  a scan.
- {{< post "hex-line-of-sight" "Line of sight and field of view on hex grids" >}}: what `hexagon` adds to the
  classic guide.
- {{< post "grid-layouts" "One API for rectangular, isometric and hexagonal grids" >}}: the map above on three
  layouts.
- {{< post "grid-pathfinding" "Pathfinding on grids" >}}: A*, Dijkstra, breadth-first and flow fields, and the one
  case where A* is wrong.
- {{< post "regions" "Regions as immutable sorted sets" >}}: the set under every area query, against a Go map.
- {{< post "errors" "Structured errors" >}} and {{< post "testing" "how the libraries are tested" >}}: the two
  smaller libraries the rest are built and tested with.
- {{< post "breaking-changes-after-v1" "Breaking changes after v1.0.0" >}}: what tagging v1 too early cost, and
  what the engine does instead.

The engine built on all of them gets a series of its own after this one.

## Installation

```shell
go get github.com/gravitton/geometry
go get github.com/gravitton/hexagon
go get github.com/gravitton/grid
```

Each README has the full usage, with figures drawn from what the calls return, and the package documentation is
the reference:
[geometry](https://pkg.go.dev/github.com/gravitton/geometry), [hexagon](https://pkg.go.dev/github.com/gravitton/hexagon)
and [grid](https://pkg.go.dev/github.com/gravitton/grid).

Every number above comes from
[the code of this post](https://github.com/gravitton/blog/tree/main/content/posts/01-gravitton-libraries/code),
pinned to `geometry v1.18.0`, `hexagon v1.8.0` and `grid v1.3.0`.

{{< post "generic-numbers" "The next post" >}} takes the first of the shared rules and measures what generics over
number types cost in Go.
