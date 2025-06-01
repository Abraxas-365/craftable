package main

import (
	"fmt"
	"time"

	"github.com/Abraxas-365/craftable/ptrx"
)

type User struct {
	Name  *string
	Age   *int
	Email *string
}

func main() {
	// Create some pointers
	name := ptrx.String("John")
	var nilName *string
	age := ptrx.Int(30)
	var nilAge *int

	// Safe dereferencing with zero values
	fmt.Printf("Name: %s\n", ptrx.StringValue(name))    // "John"
	fmt.Printf("Name: %s\n", ptrx.StringValue(nilName)) // ""
	fmt.Printf("Age: %d\n", ptrx.IntValue(age))         // 30
	fmt.Printf("Age: %d\n", ptrx.IntValue(nilAge))      // 0

	// Safe dereferencing with custom defaults
	fmt.Printf("Name: %s\n", ptrx.StringValueOr(nilName, "Unknown")) // "Unknown"
	fmt.Printf("Age: %d\n", ptrx.IntValueOr(nilAge, -1))             // -1

	// Generic functions
	fmt.Printf("Generic Name: %s\n", ptrx.Value(name))        // "John"
	fmt.Printf("Generic Name: %s\n", ptrx.Value(nilName))     // ""
	fmt.Printf("Generic Age: %d\n", ptrx.ValueOr(nilAge, 18)) // 18

	// Nil checks
	fmt.Printf("Name is nil: %v\n", ptrx.IsNil(nilName))   // true
	fmt.Printf("Age is not nil: %v\n", ptrx.IsNotNil(age)) // true

	// Working with structs
	user := User{
		Name:  ptrx.String("Alice"),
		Age:   nil, // nil pointer
		Email: ptrx.String("alice@example.com"),
	}

	// Safe access to struct fields
	fmt.Printf("User: %s, Age: %d, Email: %s\n",
		ptrx.StringValue(user.Name),  // "Alice"
		ptrx.IntValueOr(user.Age, 0), // 0 (default for nil)
		ptrx.StringValue(user.Email)) // "alice@example.com"

	// Time examples
	now := ptrx.Time(time.Now())
	var nilTime *time.Time

	fmt.Printf("Time: %v\n", ptrx.TimeValue(now))                        // current time
	fmt.Printf("Time: %v\n", ptrx.TimeValue(nilTime))                    // zero time
	fmt.Printf("Time: %v\n", ptrx.TimeValueOr(nilTime, time.Unix(0, 0))) // epoch time
}
