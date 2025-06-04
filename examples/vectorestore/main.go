package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Abraxas-365/craftable/ai/document"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/ai/splitterx"
	"github.com/Abraxas-365/craftable/ai/vstorex"
	"github.com/Abraxas-365/craftable/ai/vstorex/providers/vstorexinmemory"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Create a new OpenAI provider
	embedder := aiopenai.NewOpenAIProvider(apiKey)

	// Create a splitter for documents
	splitter, err := splitterx.NewSimpleTokenSplitter(512, 20, "text-embedding-3-small")
	if err != nil {
		log.Fatalf("Failed to create splitter: %v", err)
	}

	// Create an in-memory vector store
	inMemoryStore := vstorexinmemory.NewInMemoryStore()

	// Initialize the store with options
	err = inMemoryStore.Initialize(ctx, map[string]interface{}{
		"dimensions": 1536, // Dimension for text-embedding-3-small
	})
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	fmt.Println("In-memory store initialized successfully")

	// Create the vector store with the InMemoryStore implementation
	vectorStore := vstorex.CreateVectorStore(
		inMemoryStore,
		embedder,
		splitter,
		vstorex.WithScoreThreshold(0.0), // Set to 0 to retrieve all matches
		vstorex.WithNamespace("example"),
	)

	// Initialize the vector store
	if err := vectorStore.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize vector store: %v", err)
	}

	// Create some example documents
	docs := []document.Document{
		{
			ID:          "doc1",
			PageContent: "The quick brown fox jumps over the lazy dog",
			Metadata: map[string]interface{}{
				"source":      "example1",
				"category":    "animals",
				"original_id": "doc1",
			},
		},
		{
			ID:          "doc2",
			PageContent: "Machine learning is a subset of artificial intelligence",
			Metadata: map[string]interface{}{
				"source":      "example2",
				"category":    "technology",
				"original_id": "doc2",
			},
		},
		{
			ID:          "doc3",
			PageContent: "Neural networks are inspired by biological neurons",
			Metadata: map[string]interface{}{
				"source":      "example2",
				"category":    "technology",
				"original_id": "doc3",
			},
		},
		{
			ID:          "doc4",
			PageContent: "Paris is the capital of France and known for the Eiffel Tower",
			Metadata: map[string]interface{}{
				"source":      "example3",
				"category":    "geography",
				"original_id": "doc4",
			},
		},
	}

	// Add documents to the vector store
	addOptions := &vstorex.AddDocumentOptions{
		ProcessOptions: vstorex.ProcessOptions{
			SplitDocuments: true,
			BatchSize:      10,
		},
		GenerateIDs: false,
		AdditionalMetadata: map[string]interface{}{
			"indexed_at": "2025-05-14",
		},
	}

	if err := vectorStore.AddDocuments(ctx, docs, addOptions); err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	fmt.Println("✅ Documents added successfully")

	// Example 1: Basic similarity search
	query := "Tell me about artificial intelligence"
	searchResult, err := vectorStore.SimilaritySearch(ctx, query, &vstorex.SearchOptions{
		Limit: 2,
	})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Println("\n🔍 Search Results for:", query)
	fmt.Printf("Found %d documents (total: %d)\n", len(searchResult.Documents), searchResult.Total)
	for i, doc := range searchResult.Documents {
		fmt.Printf("%d. %s (Score: %.4f)\n", i+1, doc.PageContent, doc.Score)
		fmt.Printf("   Metadata: %v\n", doc.Metadata)
	}

	// Example 2: Search with filters
	filter := vstorex.NewFilterBuilder().
		Eq("category", "technology").
		Build()

	filteredResults, err := vectorStore.SimilaritySearch(ctx, query, &vstorex.SearchOptions{
		Limit:  10,
		Filter: filter,
	})
	if err != nil {
		log.Fatalf("Filtered search failed: %v", err)
	}

	fmt.Println("\n🔍 Technology Category Search Results for:", query)
	fmt.Printf("Found %d documents (total: %d)\n", len(filteredResults.Documents), filteredResults.Total)
	for i, doc := range filteredResults.Documents {
		fmt.Printf("%d. %s (Score: %.4f)\n", i+1, doc.PageContent, doc.Score)
		fmt.Printf("   Metadata: %v\n", doc.Metadata)
	}

	// Example 3: Complex compound filters
	compoundFilter := vstorex.CompoundFilter{
		And: []vstorex.Filter{
			vstorex.MapFilter{"source": "example2"},
			vstorex.ComparisonFilter{
				Field: "category",
				Op:    "eq",
				Value: "technology",
			},
		},
	}

	complexResults, err := vectorStore.SimilaritySearch(ctx, "neural", &vstorex.SearchOptions{
		Limit:          5,
		Filter:         compoundFilter,
		ScoreThreshold: 0.5,
	})
	if err != nil {
		log.Fatalf("Complex search failed: %v", err)
	}

	fmt.Println("\n🔍 Complex Filter Search Results:")
	fmt.Printf("Found %d documents (total: %d)\n", len(complexResults.Documents), complexResults.Total)
	for i, doc := range complexResults.Documents {
		fmt.Printf("%d. %s (Score: %.4f)\n", i+1, doc.PageContent, doc.Score)
	}

	// Example 4: Check if documents exist - using chunked IDs
	chunkIds := []string{"doc1_chunk_0", "doc2_chunk_0", "nonexistent"}
	exists, err := vectorStore.DocumentExists(ctx, chunkIds)
	if err != nil {
		log.Fatalf("Document exists check failed: %v", err)
	}

	fmt.Println("\n📝 Document Existence Check (with chunk IDs):")
	for i, id := range chunkIds {
		fmt.Printf("%s: %v\n", id, exists[i])
	}

	// Example 5: Count documents by filter
	categoryFilter := vstorex.NewFilterBuilder().Eq("category", "technology").Build()
	count, err := vectorStore.Count(ctx, categoryFilter)
	if err != nil {
		log.Fatalf("Count failed: %v", err)
	}

	fmt.Printf("\n🔢 Documents in technology category: %d\n", count)

	// Example 6: Get documents by chunked ID
	retrievedDocs, err := vectorStore.GetDocuments(ctx, []string{"doc1_chunk_0"})
	if err != nil {
		log.Fatalf("Get documents failed: %v", err)
	}

	fmt.Println("\n📄 Retrieved Documents by Chunk ID:")
	for i, doc := range retrievedDocs {
		fmt.Printf("%d. ID: %s, Content: %s\n", i+1, doc.ID, doc.PageContent)
	}

	// Example 7: Get documents using metadata-based filtering
	// USE A SIMPLE QUERY - this is the key fix
	originalIdFilter := vstorex.NewFilterBuilder().
		Eq("original_id", "doc1").
		Build()

	docsByOriginalId, err := vectorStore.SimilaritySearch(ctx, "fox", &vstorex.SearchOptions{
		Filter:         originalIdFilter,
		Limit:          10,
		ScoreThreshold: 0.0, // Return any match regardless of score
	})
	if err != nil {
		log.Fatalf("Get documents by original ID failed: %v", err)
	}

	fmt.Println("\n📄 Retrieved Documents by Original ID Metadata:")
	for i, doc := range docsByOriginalId.Documents {
		fmt.Printf("%d. ID: %s, Content: %s\n", i+1, doc.ID, doc.PageContent)
		fmt.Printf("   Metadata: %v\n", doc.Metadata)
	}

	// Example 8: Get documents by source metadata
	// USE A GENERIC QUERY STRING
	sourceFilter := vstorex.NewFilterBuilder().
		Eq("source", "example1").
		Build()

	docsBySource, err := vectorStore.SimilaritySearch(ctx, "*", &vstorex.SearchOptions{
		Filter:         sourceFilter,
		Limit:          10,
		ScoreThreshold: 0.0, // Important: return any match
	})
	if err != nil {
		log.Fatalf("Get documents by source failed: %v", err)
	}

	fmt.Println("\n📄 Retrieved Documents by Source Metadata:")
	for i, doc := range docsBySource.Documents {
		fmt.Printf("%d. ID: %s, Content: %s\n", i+1, doc.ID, doc.PageContent)
		fmt.Printf("   Metadata: %v\n", doc.Metadata)
	}

	// Example 9: Delete documents by filter
	deleteFilter := vstorex.NewFilterBuilder().Eq("category", "geography").Build()
	err = vectorStore.DeleteByFilter(ctx, deleteFilter)
	if err != nil {
		log.Fatalf("Delete by filter failed: %v", err)
	}

	// Verify deletion
	countAfterDelete, err := vectorStore.Count(ctx, nil)
	if err != nil {
		log.Fatalf("Count after delete failed: %v", err)
	}
	fmt.Printf("\n🗑️ Deleted geography documents. Remaining document count: %d\n", countAfterDelete)

	// Example 10: Using FilterBuilder with multiple clauses
	advancedFilter := vstorex.NewFilterBuilder().
		And(
			vstorex.NewFilterBuilder().Eq("source", "example2").Build(),
			vstorex.NewFilterBuilder().Contains("page_content", "neural").Build(),
		).
		Build()

	advancedResults, err := vectorStore.SimilaritySearch(ctx, "brain", &vstorex.SearchOptions{
		Limit:  5,
		Filter: advancedFilter,
	})
	if err != nil {
		log.Fatalf("Advanced search failed: %v", err)
	}

	fmt.Println("\n🔍 Advanced Filter Search Results:")
	for i, doc := range advancedResults.Documents {
		fmt.Printf("%d. %s (Score: %.4f)\n", i+1, doc.PageContent, doc.Score)
	}

	// Close the vector store
	if err := vectorStore.Close(); err != nil {
		log.Fatalf("Failed to close vector store: %v", err)
	}

	fmt.Println("\n✅ Vector store example completed successfully")
}
