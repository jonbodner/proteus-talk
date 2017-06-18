package generate_test

import (
	"fmt"
	"github.com/jonbodner/proteus-talk/generate"
	"testing"
	"time"
)

func AddSlowly(a, b int) int {
	time.Sleep(1 * time.Second)
	return a + b
}

func AddNormally(a, b int) int {
	return a + b
}

func BenchmarkAddSlowly(b *testing.B) {
	var result int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = AddSlowly(1, 2)
	}
	b.StopTimer()
	result = result
}

func BenchmarkMemoizationAddSlowly(b *testing.B) {
	memo := generate.MemoizeCalculator(AddSlowly)
	memo(1, 2)
	var result int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = memo(1, 2)
	}
	b.StopTimer()
	result = result
}

func BenchmarkAddNormally(b *testing.B) {
	var result int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = AddNormally(1, 2)
	}
	b.StopTimer()
	result = result
}

func BenchmarkMemoizationAddNormally(b *testing.B) {
	memo := generate.MemoizeCalculator(AddNormally)
	memo(1, 2)
	var result int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = memo(1, 2)
	}
	b.StopTimer()
	result = result
}

func timeThings(c generate.Calculator, a, b int) (int, time.Duration) {
	start := time.Now()
	result := c(a, b)
	end := time.Now()
	return result, end.Sub(start)
}

func TestMemoization(t *testing.T) {
	result := AddSlowly(1, 2)

	memo := generate.MemoizeCalculator(AddSlowly)
	start := time.Now()
	memoResult := memo(1, 2)
	firstCallTime := time.Now().Sub(start)

	if result != memoResult {
		t.Errorf("Result and memoResult should have been equal, but we got %v and %v", result, memoResult)
	}

	start = time.Now()
	memoResult = memo(1, 2)
	secondCallTime := time.Now().Sub(start)

	fmt.Println(firstCallTime)
	fmt.Println(secondCallTime)
	if secondCallTime > firstCallTime {
		t.Errorf("second call should be faster than first call, but wasnt: %v and %v", firstCallTime, secondCallTime)
	}
}
