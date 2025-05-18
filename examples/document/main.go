package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Abraxas-365/craftable/ai/document"
	"github.com/Abraxas-365/craftable/fsx"
	"github.com/Abraxas-365/craftable/fsx/fsxprovider"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create a temporary directory for our test files
	tempDir, err := os.MkdirTemp("", "textloader-example")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir) // Clean up at the end

	fmt.Printf("Created temporary directory at: %s\n", tempDir)

	// Initialize the local filesystem provider
	localFS := fsxprovider.NewLocalFS(tempDir)

	// Create test directory structure and files
	setupTestFiles(ctx, localFS)

	// Try loading documents one by one first to diagnose where the issue is
	fmt.Println("\nTesting individual file loading:")
	testFilePath := filepath.Join("text_files", "sample1.txt")

	// Try to get file info
	info, err := localFS.Stat(ctx, testFilePath)
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}
	fmt.Printf("File info for %s: size=%d, isDir=%v\n", testFilePath, info.Size, info.IsDir)

	// Try to read the file
	data, err := localFS.ReadFile(ctx, testFilePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	fmt.Printf("Successfully read %d bytes from %s\n", len(data), testFilePath)

	// Try creating a loader for just this file
	singleLoader := document.NewTextLoader(localFS, testFilePath, false)
	docs, err := singleLoader.Load(ctx)
	if err != nil {
		fmt.Printf("Error loading single file with TextLoader: %v\n", err)
		return
	}
	fmt.Printf("Successfully loaded single file with TextLoader, got %d document(s)\n", len(docs))

	// Create a TextLoader with the filesystem
	fmt.Println("\nLoading all documents recursively...")
	loader := document.NewTextLoader(localFS, "", true) // empty path means root, true for recursive

	// Load documents
	docs, err = loader.Load(ctx)
	if err != nil {
		fmt.Printf("Error loading documents: %v\n", err)
		return
	}

	// Print document information
	fmt.Printf("Successfully loaded %d documents:\n", len(docs))
	for i, doc := range docs {
		fmt.Printf("\nDocument %d:\n", i+1)
		fmt.Printf("  Source: %s\n", doc.Source)
		fmt.Printf("  Content length: %d characters\n", len(doc.PageContent))
		fmt.Printf("  Content preview: %s\n", truncateString(doc.PageContent, 50))
		fmt.Printf("  Metadata:\n")
		for k, v := range doc.Metadata {
			fmt.Printf("    %s: %v\n", k, v)
		}
	}
}

// setupTestFiles creates a test directory structure with various file types
func setupTestFiles(ctx context.Context, fs fsx.FileSystem) {
	// Create directory structure
	dirs := []string{
		"text_files",
		"text_files/subdirectory",
		"code_files",
		"non_text_files",
	}

	for _, dir := range dirs {
		err := fs.CreateDir(ctx, dir)
		if err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
			return
		}
	}

	// Create text files
	textFiles := map[string]string{
		"text_files/sample1.txt":               "This is a plain text file.\nIt has multiple lines.\nIt should be loaded by the TextLoader.",
		"text_files/sample2.txt":               "Another text file with different content.",
		"text_files/readme.md":                 "# Markdown File\nThis is a markdown file that should be loaded.",
		"text_files/subdirectory/nested.txt":   "This file is in a subdirectory, it should be loaded when recursive is true.",
		"code_files/example.go":                "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, world!\")\n}",
		"code_files/script.py":                 "def greet():\n    print(\"Hello from Python\")\n\ngreet()",
		"non_text_files/should_be_skipped.bin": string([]byte{0x00, 0x01, 0x02, 0x03}), // Binary content that should be skipped
	}

	for path, content := range textFiles {
		err := fs.WriteFile(ctx, path, []byte(content))
		if err != nil {
			fmt.Printf("Failed to create file %s: %v\n", path, err)
			return
		}
	}

	// Verify files were created
	for path := range textFiles {
		exists, err := fs.Exists(ctx, path)
		if err != nil || !exists {
			fmt.Printf("WARNING: File %s does not exist after creation attempt\n", path)
		}
	}

	fmt.Printf("Created test files in %s:\n", fs.Join())
	printDirectoryTree(ctx, fs, "", "")
}

// printDirectoryTree recursively prints the directory structure
func printDirectoryTree(ctx context.Context, fs fsx.FileSystem, path string, indent string) {
	var pathToList string
	if path == "" {
		pathToList = ""
		fmt.Println(".")
	} else {
		pathToList = path
		fmt.Println(indent + filepath.Base(path))
	}

	entries, err := fs.List(ctx, pathToList)
	if err != nil {
		fmt.Printf("%sError listing %s: %v\n", indent, path, err)
		return
	}

	// Sort entries to have directories first, then files
	var dirs []fsx.FileInfo
	var files []fsx.FileInfo
	for _, entry := range entries {
		if entry.IsDir {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	// Process directories
	for i, entry := range dirs {
		isLast := i == len(dirs)-1 && len(files) == 0
		newIndent := indent
		if isLast {
			newIndent += "    "
		} else {
			newIndent += "│   "
		}

		prefix := indent
		if isLast {
			prefix += "└── "
		} else {
			prefix += "├── "
		}

		entryPath := fs.Join(path, entry.Name)
		fmt.Printf("%s%s\n", prefix, entry.Name)
		printDirectoryTree(ctx, fs, entryPath, newIndent)
	}

	// Process files
	for i, entry := range files {
		isLast := i == len(files)-1
		prefix := indent
		if isLast {
			prefix += "└── "
		} else {
			prefix += "├── "
		}

		fmt.Printf("%s%s (%d bytes)\n", prefix, entry.Name, entry.Size)
	}
}

// Helper function to truncate a string
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

