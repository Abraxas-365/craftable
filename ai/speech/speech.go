package speech

import (
	"context"
	"fmt"
	"io"

	"github.com/Abraxas-365/craftable/fsx"
	"github.com/Abraxas-365/craftable/fsx/fsxprovider"
)

//---------- Text-to-Speech (TTS) ----------//

// Speaker represents an interface for text-to-speech operations
type Speaker interface {
	// Synthesize converts text to speech audio
	Synthesize(ctx context.Context, text string, opts ...SynthesisOption) (Audio, error)
}

// Audio represents the generated speech audio
type Audio struct {
	// Content is the audio data
	Content io.ReadCloser

	// Format indicates the audio format (MP3, WAV, etc.)
	Format AudioFormat

	// SampleRate of the audio in Hz
	SampleRate int

	// Usage contains token/resource usage statistics
	Usage TTSUsage
}

// TTSUsage represents resource usage statistics for text-to-speech
type TTSUsage struct {
	InputCharacters int
	ProcessingTime  int // in milliseconds
}

//---------- Speech-to-Text (STT) ----------//

// Transcriber represents an interface for speech-to-text operations
type Transcriber interface {
	// Transcribe converts speech audio to text
	Transcribe(ctx context.Context, audio io.Reader, opts ...TranscriptionOption) (Transcript, error)
}

// Transcript represents the result of a speech-to-text operation
type Transcript struct {
	// Text is the transcribed text
	Text string

	// Segments contains detailed information about segments (if supported)
	Segments []TranscriptSegment

	// LanguageCode is the detected language (if available)
	LanguageCode string

	// Confidence is the overall confidence score (0-1)
	Confidence float32

	// Usage contains token/resource usage statistics
	Usage STTUsage
}

// TranscriptSegment represents a segment of transcribed text
type TranscriptSegment struct {
	// Text is the content of this segment
	Text string

	// StartTime is the start time in seconds
	StartTime float32

	// EndTime is the end time in seconds
	EndTime float32

	// Confidence is the confidence score for this segment (0-1)
	Confidence float32
}

// STTUsage represents resource usage statistics for speech-to-text
type STTUsage struct {
	AudioDuration  float32 // in seconds
	ProcessingTime int     // in milliseconds
}

//---------- Common Types ----------//

// AudioFormat represents the format of speech audio
type AudioFormat string

const (
	AudioFormatMP3 AudioFormat = "mp3"
	AudioFormatWAV AudioFormat = "wav"
	AudioFormatPCM AudioFormat = "pcm"
	AudioFormatOGG AudioFormat = "ogg"
)

//---------- Clients ----------//

// TTSClient represents a configured text-to-speech client
type TTSClient struct {
	speaker Speaker
}

// NewTTSClient creates a new text-to-speech client
func NewTTSClient(speaker Speaker) *TTSClient {
	return &TTSClient{speaker: speaker}
}

// Synthesize converts text to speech audio
func (c *TTSClient) Synthesize(ctx context.Context, text string, opts ...SynthesisOption) (Audio, error) {
	return c.speaker.Synthesize(ctx, text, opts...)
}

// STTClient represents a configured speech-to-text client
type STTClient struct {
	transcriber Transcriber
	fs          fsx.FileReader
}

// NewSTTClient creates a new speech-to-text client with default local filesystem
func NewSTTClient(transcriber Transcriber) *STTClient {
	// Use local filesystem by default
	localFS := fsxprovider.NewLocalFS("")

	return &STTClient{
		transcriber: transcriber,
		fs:          localFS,
	}
}

// WithFileSystem configures the client to use the specified file system
func (c *STTClient) WithFileSystem(fs fsx.FileReader) *STTClient {
	c.fs = fs
	return c // Return self for method chaining
}

// Transcribe converts speech audio to text
func (c *STTClient) Transcribe(ctx context.Context, audio io.Reader, opts ...TranscriptionOption) (Transcript, error) {
	return c.transcriber.Transcribe(ctx, audio, opts...)
}

// TranscribeFile converts speech audio from a file path to text using the configured file system
func (c *STTClient) TranscribeFile(ctx context.Context, filePath string, opts ...TranscriptionOption) (Transcript, error) {
	// Use the file system (which is now guaranteed to be non-nil)
	stream, err := c.fs.ReadFileStream(ctx, filePath)
	if err != nil {
		return Transcript{}, fmt.Errorf("error opening audio file: %w", err)
	}
	defer stream.Close()

	return c.transcriber.Transcribe(ctx, stream, opts...)
}
