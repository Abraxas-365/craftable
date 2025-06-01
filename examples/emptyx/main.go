package main

import (
	"fmt"

	"github.com/Abraxas-365/craftable/emptyx"
)

func main() {
	// Generic Empty function
	fmt.Println("=== Generic Empty ===")

	// Slices
	var nilSlice []int
	emptySlice := []int{}
	nonEmptySlice := []int{1, 2, 3}

	fmt.Printf("nil slice: %v\n", emptyx.Empty(nilSlice))            // true
	fmt.Printf("empty slice: %v\n", emptyx.Empty(emptySlice))        // true
	fmt.Printf("non-empty slice: %v\n", emptyx.Empty(nonEmptySlice)) // false

	// Maps
	var nilMap map[string]int
	emptyMap := make(map[string]int)
	nonEmptyMap := map[string]int{"key": 1}

	fmt.Printf("nil map: %v\n", emptyx.Empty(nilMap))            // true
	fmt.Printf("empty map: %v\n", emptyx.Empty(emptyMap))        // true
	fmt.Printf("non-empty map: %v\n", emptyx.Empty(nonEmptyMap)) // false

	// Type-specific functions (better performance)
	fmt.Println("\n=== Type-specific functions ===")

	fmt.Printf("emptyx.Slice(nilSlice): %v\n", emptyx.Slice(nilSlice))
	fmt.Printf("emptyx.Map(nilMap): %v\n", emptyx.Map(nilMap))
	fmt.Printf("emptyx.String(\"\"): %v\n", emptyx.String(""))
	fmt.Printf("emptyx.Pointer(nilPtr): %v\n", emptyx.Pointer((*int)(nil)))

	// Semantic aliases
	fmt.Println("\n=== Semantic aliases ===")
	fmt.Printf("emptyx.Nil(nilSlice): %v\n", emptyx.Nil(nilSlice))
	fmt.Printf("emptyx.Zero(0): %v\n", emptyx.Zero(0))
}
