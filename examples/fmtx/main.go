package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Abraxas-365/craftable/fmtx"
)

type Person struct {
	ID       int       `json:"id" db:"person_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Age      int       `json:"age"`
	Birthday time.Time `json:"birthday"`
	private  string    // unexported field
}

type Company struct {
	Name      string
	Employees []Person
	Founded   time.Time
	Metadata  map[string]any
}

func main() {
	people := []Person{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Age: 30, Birthday: time.Now().AddDate(-30, 0, 0)},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Age: 25, Birthday: time.Now().AddDate(-25, 0, 0)},
		{ID: 3, Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Birthday: time.Now().AddDate(-35, 0, 0)},
	}

	company := Company{
		Name:      "Tech Corp",
		Employees: people,
		Founded:   time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
		Metadata: map[string]any{
			"stock_symbol": "TECH",
			"revenue":      1000000.50,
			"public":       true,
		},
	}

	// Basic debug
	fmt.Println("=== Basic Debug ===")
	fmtx.DebugPrint(company)

	// Pretty print with colors
	fmt.Println("\n=== Pretty Print ===")
	fmtx.PrettyPrint(company)

	// Compact format
	fmt.Println("\n=== Compact ===")
	fmtx.CompactPrint(company)

	// JSON-like format
	fmt.Println("\n=== JSON-like ===")
	fmtx.JSONPrint(company)

	// Table format for slice
	fmt.Println("\n=== Table Format ===")
	fmtx.TablePrint(people)

	// Custom options
	fmt.Println("\n=== Custom Options ===")
	opts := fmtx.DefaultOptions()
	opts.UseColors = true
	opts.ShowTypes = true
	opts.ShowSizes = true
	opts.ShowPrivate = true
	opts.MaxStringLength = 30

	// Custom time formatter
	opts.CustomFormatters = make(map[reflect.Type]func(reflect.Value) string)
	opts.CustomFormatters[reflect.TypeOf(time.Time{})] = fmtx.TimeFormatter

	result := fmtx.DebugWithOptions(company, opts)
	fmt.Println(result)

	// Field filtering - only fields with json tags
	fmt.Println("\n=== JSON Fields Only ===")
	jsonOpts := fmtx.WithFieldFilter(fmtx.FieldsWithTag("json"))
	fmt.Println(fmtx.DebugWithOptions(people[0], jsonOpts))

	// Size information
	fmt.Println("\n=== Size Info ===")
	fmt.Println(fmtx.Size(company))

	// Diff two values
	fmt.Println("\n=== Diff ===")
	person1 := people[0]
	person2 := person1
	person2.Age = 31
	person2.Email = "john.doe@example.com"
	fmtx.DiffPrint(person1, person2)

	// Hexdump
	fmt.Println("\n=== Hexdump ===")
	data := []byte("Hello, World! This is a test of hexdump functionality.")
	fmtx.HexdumpPrint(data)

	// Stack trace
	fmt.Println("\n=== Stack Trace ===")
	fmtx.StackPrint()

	// Timing
	fmt.Println("\n=== Timing ===")
	timer := fmtx.StartTimer("Processing")
	time.Sleep(100 * time.Millisecond)
	timer.StopAndPrint()

	// Memory addresses
	fmt.Println("\n=== Memory Info ===")
	fmt.Printf("Company address: %s\n", fmtx.MemoryAddress(&company))
	fmt.Printf("Name string info: %s\n", fmtx.UnsafeString(company.Name))
	fmt.Printf("Employees slice info: %s\n", fmtx.UnsafeSlice(company.Employees))
}
