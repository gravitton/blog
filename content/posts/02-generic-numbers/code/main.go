//lint:file-ignore U1000 benchmark results go to sinks that are never read, so the compiler keeps the work

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"

	geom "github.com/gravitton/geometry"
)

type vector struct {
	X, Y float64
}

func (v vector) cross(w vector) float64 {
	return float64(v.X*w.Y) - float64(v.Y*w.X)
}

func (v vector) multiply(factor float64) vector {
	return vector{float64(v.X * factor), float64(v.Y * factor)}
}

var (
	sinkFloat  float64
	sinkVector geom.Vector[float64]
	sinkInts   geom.Vector[int]
	sinkPlain  vector
)

func main() {
	profile := flag.String("cpuprofile", "", "write a CPU profile of the measured work")
	flag.Parse()
	if *profile != "" {
		file, err := os.Create(*profile)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		if err := pprof.StartCPUProfile(file); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	for _, m := range []struct {
		name  string
		bench func(*testing.B)
	}{
		{"cross, plain struct", crossPlain},
		{"cross, geom.Vector[float64]", crossGeneric},
		{"multiply, plain struct", multiplyPlain},
		{"multiply, geom.Vector[float64]", multiplyFloat},
		{"multiply, geom.Vector[int]", multiplyInt},
	} {
		result := testing.Benchmark(m.bench)
		fmt.Printf("%-32s %5.2f ns a vector\n", m.name, float64(result.T.Nanoseconds())/float64(result.N)/count)
	}
}

const count = 1024

func plainVectors() []vector {
	vectors := make([]vector, count)
	for i := range vectors {
		vectors[i] = vector{float64(i%37) * 0.25, float64(i%41) * -0.5}
	}

	return vectors
}

func geomVectors() []geom.Vector[float64] {
	vectors := make([]geom.Vector[float64], count)
	for i, v := range plainVectors() {
		vectors[i] = geom.Vec(v.X, v.Y)
	}

	return vectors
}

func intVectors() []geom.Vector[int] {
	vectors := make([]geom.Vector[int], count)
	for i := range vectors {
		vectors[i] = geom.Vec(i%37, -(i % 41))
	}

	return vectors
}

func crossPlain(bench *testing.B) {
	vectors := plainVectors()
	for range bench.N {
		sum := 0.0
		for i := 1; i < len(vectors); i++ {
			sum += vectors[i].cross(vectors[i-1])
		}
		sinkFloat = sum
	}
}

func crossGeneric(bench *testing.B) {
	vectors := geomVectors()
	for range bench.N {
		sum := 0.0
		for i := 1; i < len(vectors); i++ {
			sum += vectors[i].Cross(vectors[i-1])
		}
		sinkFloat = sum
	}
}

func multiplyPlain(bench *testing.B) {
	vectors := plainVectors()
	for range bench.N {
		sum := vector{}
		for _, v := range vectors {
			m := v.multiply(1.5)
			sum = vector{sum.X + m.X, sum.Y + m.Y}
		}
		sinkPlain = sum
	}
}

func multiplyFloat(bench *testing.B) {
	vectors := geomVectors()
	for range bench.N {
		sum := geom.Vector[float64]{}
		for _, v := range vectors {
			sum = sum.Add(v.Multiply(1.5))
		}
		sinkVector = sum
	}
}

func multiplyInt(bench *testing.B) {
	vectors := intVectors()
	for range bench.N {
		sum := geom.Vector[int]{}
		for _, v := range vectors {
			sum = sum.Add(v.Multiply(1.5))
		}
		sinkInts = sum
	}
}
