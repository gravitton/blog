---
title: "What generics over number types cost in Go"
date: 2026-11-10
slug: generic-numbers
draft: false
description: "A one-line generic method costs four times its hand-written twin, because the inliner prices a branch the compiler deletes later. Profile-guided optimization takes all of it back."
repository: geometry
keywords: ["Go generics", "performance", "inlining", "profile-guided optimization", "PGO", "GC shape", "geometry"]
---

[`gravitton/geometry`](https://github.com/gravitton/geometry) is generic over its number type. A `Point[int]` is
a tile, a `Point[float64]` is a position, and both have the same methods, written once:

```go
type Number interface {
	Integer | Float // ~int, ~int8 … ~int64, ~float32 and ~float64
}

type Vector[T Number] struct {
	X, Y T
}
```

It is the first of the rules the [three libraries share](../gravitton-libraries/), and the one that is easiest to
doubt: Go's generics are often said to be slow, and a geometry library is called from the hottest loops a game has.
This post measures what the rule costs, finds the one place it is not free, and shows the compiler flag that takes
it back.

## One branch per instantiation

A generic method that scales a vector has to compute in `float64` and store the result back into `T`, rounding for
an integer `T` and not for a float one. Every such method ends in `Cast`:

```go
func Cast[T Number](value float64) T {
	if isInt[T]() {
		if !IsFinite(value) {
			panic(ErrNonFinite)
		}

		return T(math.Round(value))
	}

	return T(value)
}

func isInt[T Number]() bool {
	return T(1)/T(2) == 0
}
```

`isInt` needs no reflection and no `unsafe`: one half is zero in integer division and a half in floating point. For
each instantiation it is a constant, and the compiler deletes the branch it can never take. The `float64` body of
`Vector.Multiply` compiled for arm64 contains no rounding instruction at all; the `int` body contains two
`FRINTA`, one per coordinate. At run time, the integer branch costs a float vector nothing.

## Where it is not free

Then measure it. Summing over 1 024 vectors on an Apple M3, against a plain struct with the same methods written by
hand:

| Per vector | Plain struct | `geom.Vector[float64]` |
|---|---|---|
| `Cross` | 0.74 ns | 0.88 ns |
| `Multiply` | 0.73 ns | 3.2 ns |

`Multiply` costs four times its hand-written twin. Nothing in it is slow; the difference is a function call. The
plain struct's method is inlined into the loop, and the generic one is called. Ask the compiler why, with
`-gcflags=-m=2`:

```
cannot inline Vector[go.shape.float64].Multiply: function too complex: cost 120 exceeds budget 80
```

A one-line method costs 120 against a budget of 80, and the reason is the branch from above. The inliner prices a
function before the branch is deleted, so the `float64` instantiation pays for the `math.Round`, the finiteness
check and the panic it will never run. `Cast` itself is cheap enough on arm64, at a cost of 38, but every method
that stores two coordinates calls it twice, and that is enough to cross the line. `Cross` calls it once and still
comes to 97, which is the smaller gap in its row.

Go compiles generics by GC shape: one body per underlying type, with a dictionary passed in for what differs. The
method a caller sees is a small wrapper, cheap enough to inline at a cost of 63, around that shared body, which is
the part over budget. So the wrapper is inlined, and the call it wraps is not.

## Priced higher on amd64

On amd64 the inliner is stricter still, for a reason that has nothing to do with generics. `math.Round` rounds
half away from zero, and x86 has no instruction for that rounding; the SSE4.1 `ROUNDSD` instruction rounds half to
even, toward zero, up or down, but not half away. So the Go compiler turns `math.Round` into a single instruction
on arm64, `FRINTA`, and on ppc64 and s390x, and on amd64 calls the function, a dozen operations on the bits of the
float.

The inliner sees that too. On amd64, `Cast` costs 109 instead of 38, over the budget on its own:

```
cannot inline Cast[go.shape.float64]: function too complex: cost 109 exceeds budget 80
```

Note the type: `float64`. The instantiation whose body is a plain conversion is not inlined on amd64, because its
cost includes a rounding function it never calls. A float method on amd64 pays a call for `Cast` inside the call
for the method.

## What profile-guided optimization takes back

The budget of 80 is a guess the compiler makes without knowing which calls matter. Profile-guided optimization
gives it the answer. Record a CPU profile of a representative run and put it next to `main.go` as `default.pgo`;
since Go 1.21 `go build` uses it automatically, and raises the budget for the call sites the profile marks hot.

```shell
go run -pgo=off . -cpuprofile default.pgo   # record the profile
go run .                                     # build with it
```

With the profile, the compiler inlines the generic `Multiply` and `Cross` into the loop:

| Per vector | Plain struct | `geom.Vector[float64]`, no profile | `geom.Vector[float64]`, `default.pgo` |
|---|---|---|---|
| `Cross` | 0.74 ns | 0.88 ns | 0.73 ns |
| `Multiply` | 0.73 ns | 3.2 ns | 0.74 ns |

The generic version is now as fast as the hand-written one. The package documentation says this in two sentences,
under Performance: methods that store two coordinates through `Cast` exceed the inlining budget, and a
profile-guided build inlines them where the profile marks them hot.

## The options

There are three ways to answer it.

- **Write the hot methods per type.** `Point[float64].Multiply` by hand, generic for the rest. It doubles the code
  that has to agree, in a library whose whole point is that methods agree, and Go has no specialization to make it
  automatic.
- **Make `Cast` cheaper to price.** The panic and the finiteness check are the guard against a NaN becoming an
  arbitrary integer; removing them trades a correct answer for a cheaper one in every integer method.
- **Leave the code generic and document the profile.** A game is exactly the program that has a representative
  run to profile, and the methods it calls in a hot loop are exactly the ones the profile finds.

The library takes the third. The cost is real, measured, and a flag away from zero, and the generic code stays one
copy.

## Closing

Generics over number types in Go are free at run time and not free at compile time: the inliner prices every
branch of every instantiation before the dead ones are removed. Profile-guided optimization is the tool for exactly
that gap, and it is worth turning on before writing a single specialized copy.

The type parameter decides one more thing than the rounding in `Cast`: how close to an edge a point can be and
still count as inside, which is {{< post "closed-boundaries" "the next post" >}}.

Every number above comes from
[the code of this post](https://github.com/gravitton/blog/tree/main/content/posts/02-generic-numbers/code), pinned to
`geometry v1.18.0`, measured on an Apple M3 with Go 1.27.3; its `default.pgo` is the profile it records. The amd64
inlining costs are the compiler's own, from `GOARCH=amd64 go build -gcflags=-m=2`; the timings were not repeated on
amd64.
