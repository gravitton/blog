---
title: "Closed boundaries within a tolerance in a Go geometry library"
date: 2026-11-17
slug: closed-boundaries
draft: false
description: "Build a float rectangle from two corners and ask whether it contains them. With exact comparison, 63% of the time it does not."
repository: geometry
keywords: ["Go", "floating point", "tolerance", "epsilon", "closed boundary", "point in rectangle", "collision", "NaN", "geometry"]
---

Build a rectangle from two corners, then ask whether it contains them. It is the kind of question a test answers
once and then nobody asks again:

```go
low := geom.Pt(851.43, 30.29)
high := geom.Pt(1274.32, 989.16)
rectangle := geom.RectangleFromMinMax(low, high)

rectangle.TopLeft() // (851.4300000000001, 30.29000000000002)
```

The corner it gives back is not the corner it was given. Compare them with `<=`, the way most code would, and the
rectangle does not contain its own corner. That is not a rare unlucky pair. Out of a million rectangles built from
random corners:

| | Rectangles |
|---|---|
| `TopLeft()` or `BottomRight()` is not bit-identical to the corner given | 894 538 |
| A given corner is outside by exact `<=` | 626 475 |
| Both given corners inside by `Contains` | 1 000 000 |

The last row is what [`gravitton/geometry`](https://github.com/gravitton/geometry) answers today. This post is
about the rule behind it, the second of the [rules the libraries share](../gravitton-libraries/), what it costs,
and where it is still wrong.

## Why the corner moves

A `Rectangle` is stored as a center, a size and an angle:

```go
type Rectangle[T Number] struct {
	Center Point[T]
	Size   Size[T]
	Angle  float64
}
```

`RectangleFromMinMax` turns the two corners into a center `low + (high-low)/2` and a size `high-low`, and `TopLeft`
turns them back into `center - size/2`, through the rotation. Each step rounds. The corner above comes back one ulp
to the right and a little lower: `1.1e-13` and `2.1e-14`, nothing a player could ever see, and enough to fail `<=`.

Why not store the corners and have no problem? Because a rectangle that turns has no "min" and "max" that stay
corners, anchoring and scaling work around the center, and `Translate` should move one field, not two. For the
axis-aligned case where the corners are what you have, the library has `Box`, which stores `Min` and `Max` and
returns them untouched. A `Rectangle` gives that up for the turn, and the rounding is the price.

## Three answers a library can give

Once a coordinate can be one rounding off, "is this point inside?" has three possible answers at the edge.

1. **Half-open, `[min, max)`.** This is `image.Rectangle` in the standard library, and it is right for pixels: a
   rectangle of width `w` covers exactly `w` columns, and two rectangles side by side share none. It is wrong for
   collision. Two bodies resting against each other are not touching, and a ball that lands exactly on the floor
   falls through it.
2. **Closed and exact, `[min, max]`.** Correct on paper. In floats it is the table above: the edge is inside only
   when the arithmetic that produced it happened not to round.
3. **Closed within a tolerance.** The edge is inside, and so is a point a rounding error away from it.

Before v1.13 the library did not pick one. `Rectangle.Contains` was closed and exact, and `Circle.Contains` was
open, so the edge of a rectangle was inside and the edge of a circle was not. Version 1.13 made every boundary the
third kind. That changed answers callers relied on, and it shipped as a minor release with the change marked
**breaking** in the [release notes](https://github.com/gravitton/geometry/blob/main/docs/releases/v1.13.0.md); why
it was not a v2 is {{< post "breaking-changes-after-v1" "the last post of the series" >}}.

## The rule

Every `Contains` and `Intersects` in the library is closed, and a point within a tolerance of a boundary is on it.
The tolerance depends only on the coordinate type, the type parameter of the
[previous post](../generic-numbers/):

| `T` | Tolerance near the origin | Far from the origin |
|---|---|---|
| any integer | 0, compared exactly | 0 |
| `float32` | `1e-4` | two to four ulps of `float32` |
| `float64` | `1e-6` | two to four ulps of `float64` |

```go
rectangle := geom.RectangleFromMinMax(geom.Pt(0.0, 0.0), geom.Pt(10.0, 10.0))

rectangle.Contains(geom.Pt(10+5e-7, 5.0))   // true
rectangle.DistanceTo(geom.Pt(10+5e-7, 5.0)) // 0
rectangle.Contains(geom.Pt(10+5e-6, 5.0))   // false
rectangle.DistanceTo(geom.Pt(10+5e-6, 5.0)) // 4.999999999810711e-06
```

The constant matters less than the invariant in the second line: `DistanceTo` is zero exactly where `Contains` is
true. Two methods that disagree by one rounding at the boundary are worse than either being slightly wrong, because
the bug shows up only where a caller combines them: a point that is inside but at a distance of `1e-13`, so the
code that pushes a body out by its distance moves it by nothing, forever. The library gets the agreement by
construction rather than by testing it into shape: `Rectangle.Contains` is one line,

```go
return r.DistanceSquaredTo(point) == 0
```

and `DistanceSquaredTo` is the one place the tolerance is applied. Every shape works the same way, so there is one
comparison per boundary and nothing for two methods to disagree about.

## Touching shapes collide

```go
left := geom.RectangleFromMinMax(geom.Pt(0.0, 0.0), geom.Pt(10.0, 10.0))
right := geom.RectangleFromMinMax(geom.Pt(10.0, 0.0), geom.Pt(20.0, 10.0))

left.IntersectsRectangle(right) // true
```

Anyone coming from tile maths, where neighbouring tiles share no pixel, will read this as a bug. It is the point of
a closed boundary: a crate standing on a floor touches it. A caller who wants strict overlap asks for the common
part and looks at its area. `left.ClipRectangle(right)` gives a rectangle `0×10` with an area of zero, and zero
area is the half-open answer, stated explicitly where it is wanted. What a touch gives a body resting on a floor,
and what a half-open edge would cost it every frame, is in {{< post "collisions" "the post on collisions" >}}.

## NaN is inside nothing

```go
point := geom.Pt(math.NaN(), 5.0)

!(point.X < 0 || point.X > 10 || point.Y < 0 || point.Y > 10) // true
rectangle.Contains(point)                                    // false
```

This one was a real bug, fixed in v1.14. A point with a NaN coordinate was inside every rectangle and every
polygon, because "inside" was written as "not outside", and every comparison with NaN is false. So "not outside"
was true for a point that is nowhere. It is the kind of bug that never shows in a test, because nobody writes a
test with NaN in it, and always shows in a game, because a division by a zero-length vector is one frame away.
Since v1.14 a point with a NaN coordinate is inside no shape, and every loop in the library ends on one.

## What it costs

A tolerant comparison is a subtraction and a comparison. Measured on an Apple M3 with Go 1.27.4 and
`geometry v1.18.0`, with no allocations anywhere:

| Benchmark | ns/op |
|---|---|
| `EqualDelta`, tolerance passed in | 0.33 |
| `Equal`, tolerance from the type | 0.36 |
| `EqualRelative`, tolerance scaled with the magnitude | about 2 |
| `Rectangle.IntersectsRectangle` | 7.2 |
| `Segment.IntersectsRectangle`, apart | 8.3 |
| `Segment.IntersectsRectangle`, through | 46.2 |
| `Polygon.Contains`, outside its extent | 28.3 |
| `Polygon.Contains`, inside, 64 edges | about 470 |

Looking the tolerance up from the type costs almost nothing on top of the comparison, which is why `Equal` is the
default and every shape uses it. A tolerance that scales with the values, `EqualRelative`, costs about five times
as much. It exists for values far from the origin where an absolute tolerance is smaller than one ulp, and no shape
reaches for it; the boundary instead widens to a few ulps far out, which keeps the cheap comparison.

## Where it is still wrong

A rule that claims to fix floating point is lying, so here is what this one does not do.

- **A tolerant comparison is not transitive.** `a ≈ b` and `b ≈ c` do not give `a ≈ c`, so a tolerance cannot
  order anything. `Point.Compare` is exact for that reason, which means two points `Equal` calls the same can still
  sort apart.
- **Two methods can still disagree** within the tolerance of a boundary, when they place the same point through
  different roundings, as a turned shape, a ray or a line does. Within the supported range no answer is wrong by
  more than the tolerance, but two answers can differ by it.
- **The supported range is narrower than the type.** `float32` is tested within `1e5` of the origin, and integer
  signs are exact while coordinate differences stay within `2^26`.

All three are written down in the package documentation, which is where a caller looks when an answer surprises
them.

## Tests that use the same tolerance

A test of a tolerant library needs the same tolerance, or it fails on the library's correct answers. The
`geomtest` package asserts shapes within exactly the comparison the library uses, so a test and the code under it
agree on what "equal" means. It is built on [`gravitton/assert`](https://github.com/gravitton/assert) rather than
testify, because pulling `go-spew`, `go-difflib`, `yaml.v3` and `objx` into a geometry library for a test helper is
the wrong trade, and a user's `go.mod` gains nothing either way: `go mod tidy` in a module using geometry requires
geometry alone. The rest of how the libraries are tested is {{< post "testing" "a post of its own" >}}.

## One tolerance, asked one way

The edge is inside, within one tolerance chosen by the type, and every method that asks the question asks it the
same way. The other half of "the answer does not depend on how you ask" is that it does not depend on where you
ask, which is {{< post "fused-multiply-add" "the next post: the same bits on amd64 and arm64" >}}.

Every number above comes from
[the code of this post](https://github.com/gravitton/blog/tree/main/content/posts/03-closed-boundaries/code), pinned to
`geometry v1.18.0`; the timings were not repeated on amd64.
