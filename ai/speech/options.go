package speech

//---------- Text-to-Speech Options ----------//

// SynthesisOption represents a configuration option for text-to-speech operations
type SynthesisOption func(*SynthesisOptions)

// SynthesisOptions contains all configurable parameters for text-to-speech operations
type SynthesisOptions struct {
	Voice       string
	Model       string
	SpeechRate  float32 // 1.0 is normal speed
	AudioFormat AudioFormat
	SampleRate  int
}

// WithVoice sets the voice to use
func WithVoice(voice string) SynthesisOption {
	return func(o *SynthesisOptions) {
		o.Voice = voice
	}
}

// WithTTSModel sets the TTS model to use
func WithTTSModel(model string) SynthesisOption {
	return func(o *SynthesisOptions) {
		o.Model = model
	}
}

// WithSpeechRate sets the speech rate multiplier
func WithSpeechRate(rate float32) SynthesisOption {
	return func(o *SynthesisOptions) {
		o.SpeechRate = rate
	}
}

// WithOutputFormat sets the audio output format
func WithOutputFormat(format AudioFormat) SynthesisOption {
	return func(o *SynthesisOptions) {
		o.AudioFormat = format
	}
}

// WithOutputSampleRate sets the audio sample rate in Hz
func WithOutputSampleRate(sampleRate int) SynthesisOption {
	return func(o *SynthesisOptions) {
		o.SampleRate = sampleRate
	}
}

//---------- Speech-to-Text Options ----------//

// TranscriptionOption represents a configuration option for speech-to-text operations
type TranscriptionOption func(*TranscriptionOptions)

// TranscriptionOptions contains all configurable parameters for speech-to-text operations
type TranscriptionOptions struct {
	Model       string
	Language    string
	Timestamps  bool
	Diarization bool // speaker identification
	AudioFormat AudioFormat
}

// WithSTTModel sets the STT model to use
func WithSTTModel(model string) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Model = model
	}
}

// WithLanguage sets the expected language of the audio
func WithLanguage(language string) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Language = language
	}
}

// WithTimestamps enables word/phrase timestamps in the output
func WithTimestamps(enable bool) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Timestamps = enable
	}
}

// WithDiarization enables speaker identification
func WithDiarization(enable bool) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.Diarization = enable
	}
}

// WithInputFormat specifies the format of the input audio
func WithInputFormat(format AudioFormat) TranscriptionOption {
	return func(o *TranscriptionOptions) {
		o.AudioFormat = format
	}
}

