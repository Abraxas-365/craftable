package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Abraxas-365/craftable/fsx/fsxprovider"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Create S3FileSystem
	fs := fsxprovider.NewS3FileSystem(client, "my-bucket", "my-prefix")

	// Use the file system
	ctx := context.Background()

	// List files
	files, err := fs.List(ctx, "/")
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}

	for _, file := range files {
		fmt.Printf("File: %s, Size: %d, IsDir: %v\n", file.Name, file.Size, file.IsDir)
	}

	// Write a file
	err = fs.WriteFile(ctx, "/example.txt", []byte("Hello, S3!"))
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	// Read a file
	data, err := fs.ReadFile(ctx, "/example.txt")
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	fmt.Printf("File content: %s\n", string(data))
}
