package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Abraxas-365/craftable/fsx/providers/fsxlocal"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Define a test directory where we'll create and process files
	// Using a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fsx-example")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	// Create a local filesystem provider
	localFS := fsxlocal.NewLocalFS(tempDir)

	// Create a test document
	testContent := []byte("This is a test document for the fsx example.\nIt demonstrates how to use the LocalFS provider.")
	testFilePath := "test-docs/example.txt"

	// First, create the directory
	err = localFS.CreateDir(ctx, "test-docs")
	if err != nil {
		fmt.Printf("Failed to create directory: %v\n", err)
		return
	}

	// Then write the file
	err = localFS.WriteFile(ctx, testFilePath, testContent)
	if err != nil {
		fmt.Printf("Failed to write file: %v\n", err)
		return
	}

	fmt.Println("Successfully created test file:", testFilePath)

	// Demonstrate reading file content
	content, err := localFS.ReadFile(ctx, testFilePath)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		return
	}
	fmt.Println("File content:")
	fmt.Println(string(content))

	// Demonstrate listing files in a directory
	files, err := localFS.List(ctx, "test-docs")
	if err != nil {
		fmt.Printf("Failed to list directory: %v\n", err)
		return
	}

	fmt.Println("\nFiles in directory:")
	for _, file := range files {
		fmt.Printf("- %s (%d bytes, dir: %v)\n", file.Name, file.Size, file.IsDir)
	}

	// Demonstrate checking if a file exists
	exists, err := localFS.Exists(ctx, testFilePath)
	if err != nil {
		fmt.Printf("Error checking file existence: %v\n", err)
		return
	}
	fmt.Printf("\nFile exists: %v\n", exists)

	// Demonstrate getting file information
	fileInfo, err := localFS.Stat(ctx, testFilePath)
	if err != nil {
		fmt.Printf("Failed to get file info: %v\n", err)
		return
	}
	fmt.Printf("File info: name=%s, size=%d, lastModified=%v\n",
		fileInfo.Name, fileInfo.Size, fileInfo.ModTime)

	// Example of how to process multiple files
	fmt.Println("\nDemonstrating document processing workflow:")

	// Create a few more test files
	for i := 1; i <= 3; i++ {
		filePath := filepath.Join("test-docs", fmt.Sprintf("document%d.txt", i))
		content := []byte(fmt.Sprintf("Content of document %d", i))

		if err := localFS.WriteFile(ctx, filePath, content); err != nil {
			fmt.Printf("Failed to create test file %s: %v\n", filePath, err)
			return
		}
	}

	// List all files to process
	docFiles, err := localFS.List(ctx, "test-docs")
	if err != nil {
		fmt.Printf("Failed to list documents: %v\n", err)
		return
	}

	// Process each file
	fmt.Println("Processing files:")
	for _, file := range docFiles {
		if file.IsDir {
			continue // Skip directories
		}

		filePath := filepath.Join("test-docs", file.Name)

		// Check if file exists and is readable
		exists, err := localFS.Exists(ctx, filePath)
		if err != nil {
			fmt.Printf("Error checking file %s: %v\n", file.Name, err)
			continue
		}

		if !exists {
			fmt.Printf("File doesn't exist: %s\n", file.Name)
			continue
		}

		// Read file content
		content, err := localFS.ReadFile(ctx, filePath)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", file.Name, err)
			continue
		}

		fmt.Printf("Successfully processed %s (%d bytes)\n", file.Name, len(content))
	}

	fmt.Println("\nExample completed successfully")
}
