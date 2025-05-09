package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Abraxas-365/craftale/ai/aiproviders"
	"github.com/Abraxas-365/craftale/ai/embedding"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Create a new OpenAI provider
	provider := aiproviders.NewOpenAIProvider(apiKey)

	// Create a context
	ctx := context.Background()

	// Example 1: Document Embeddings
	fmt.Println("=== Document Embeddings ===")

	documents := []string{
		"The quick brown fox jumps over the lazy dog",
		"Paris is the capital of France",
		"Machine learning is a subset of artificial intelligence",
	}

	embeddings, err := provider.EmbedDocuments(
		ctx,
		documents,
		embedding.WithModel("text-embedding-3-small"),
		embedding.WithDimensions(1536),
	)
	if err != nil {
		log.Fatalf("Error generating embeddings: %v", err)
	}

	fmt.Printf("Generated %d embeddings\n", len(embeddings))
	fmt.Printf("First embedding vector length: %d\n", len(embeddings[0].Vector))
	fmt.Printf("First few vector values: %.6f, %.6f, %.6f...\n",
		embeddings[0].Vector[0],
		embeddings[0].Vector[1],
		embeddings[0].Vector[2])
	fmt.Printf("Usage: %+v\n\n", embeddings[0].Usage)

	// Example 2: Query Embedding
	fmt.Println("=== Query Embedding ===")

	query := "What is artificial intelligence?"
	queryEmbedding, err := provider.EmbedQuery(ctx, query, embedding.WithModel("text-embedding-3-small"))
	if err != nil {
		log.Fatalf("Error generating query embedding: %v", err)
	}

	fmt.Printf("Query embedding vector length: %d\n", len(queryEmbedding.Vector))
	fmt.Printf("First few vector values: %.6f, %.6f, %.6f...\n",
		queryEmbedding.Vector[0],
		queryEmbedding.Vector[1],
		queryEmbedding.Vector[2])
	fmt.Printf("Usage: %+v\n\n", queryEmbedding.Usage)

	// Example 3: Using the embedding client
	fmt.Println("=== Using Embedding Client ===")

	embeddingClient := embedding.NewClient(provider)

	// Document embeddings through client
	clientDocEmbeddings, err := embeddingClient.EmbedDocuments(
		ctx,
		documents,
		embedding.WithModel("text-embedding-3-large"),
	)
	if err != nil {
		log.Fatalf("Error using embedding client for documents: %v", err)
	}

	fmt.Printf("Generated %d embeddings using client\n", len(clientDocEmbeddings))

	// Query embedding through client
	clientQueryEmbedding, err := embeddingClient.EmbedQuery(
		ctx,
		query,
		embedding.WithModel("text-embedding-3-large"),
	)
	if err != nil {
		log.Fatalf("Error using embedding client for query: %v", err)
	}

	fmt.Printf("Successfully generated query embedding using client\n")
	fmt.Printf("Vector dimensions: %d\n", len(clientQueryEmbedding.Vector))
}
