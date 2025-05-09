package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Abraxas-365/craftale/configx"
)

func main() {
	// Set environment variables programmatically
	os.Setenv("APP_SERVER_PORT", "9000")
	os.Setenv("APP_SERVER_HOST", "api.example.com")
	os.Setenv("APP_DATABASE_URL", "postgres://user:pass@db.example.com:5432/mydb")
	os.Setenv("APP_API_KEY", "my-secret-api-key")
	os.Setenv("APP_DEBUG", "true")
	os.Setenv("APP_MAX_CONNECTIONS", "100")

	// Create configuration from environment variables
	config, err := configx.NewBuilder().
		WithDefaults(map[string]interface{}{
			"server": map[string]interface{}{
				"port": 8080,
				"host": "localhost",
			},
			"debug": false,
		}).
		FromEnv("APP_").
		RequireEnv("APP_DATABASE_URL", "APP_API_KEY").
		Build()

	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	// Print all configuration
	fmt.Println("=== All Configuration ===")
	printMap(config.AllSettings(), "")

	// Access values loaded from environment variables
	serverPort := config.Get("server.port").AsInt()
	serverHost := config.Get("server.host").AsString()
	dbURL := config.Get("database.url").AsString()
	apiKey := config.Get("api.key").AsString()
	debug := config.Get("debug").AsBool()
	maxConn := config.Get("max.connections").AsInt()

	// Print values to verify they were loaded correctly
	fmt.Println("\n=== Configuration from Environment Variables ===")
	fmt.Printf("Server:          %s:%d\n", serverHost, serverPort)
	fmt.Printf("Database URL:    %s\n", dbURL)
	fmt.Printf("API Key:         %s\n", apiKey)
	fmt.Printf("Debug Mode:      %t\n", debug)
	fmt.Printf("Max Connections: %d\n", maxConn)

	// Test missing required environment variable
	os.Unsetenv("APP_API_KEY")
	_, err = configx.NewBuilder().
		RequireEnv("APP_DATABASE_URL", "APP_API_KEY").
		Build()

	if err != nil {
		fmt.Println("\n=== Expected Error ===")
		fmt.Println(err)
	}
}

// Helper function to print the entire configuration map
func printMap(m map[string]interface{}, prefix string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if nestedMap, ok := v.(map[string]interface{}); ok {
			printMap(nestedMap, key)
		} else {
			fmt.Printf("%s = %v (%T)\n", key, v, v)
		}
	}
}

