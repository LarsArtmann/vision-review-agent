package vision

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadImageFromBase64(t *testing.T) {
	t.Parallel()

	t.Run("standard encoding", func(t *testing.T) {
		t.Parallel()
		data := []byte("fake image data")
		b64 := base64.StdEncoding.EncodeToString(data)

		img, err := LoadImageFromBase64(b64, MediaTypePNG, "test.png")
		require.NoError(t, err)
		require.Equal(t, data, img.Data)
		require.Equal(t, MediaTypePNG, img.MediaType)
		require.Equal(t, "test.png", img.Filename)
	})

	t.Run("url-safe encoding", func(t *testing.T) {
		t.Parallel()
		data := []byte("url-safe data")
		b64 := base64.URLEncoding.EncodeToString(data)

		img, err := LoadImageFromBase64(b64, MediaTypeJPEG, "img.jpg")
		require.NoError(t, err)
		require.Equal(t, data, img.Data)
	})

	t.Run("raw encoding without padding", func(t *testing.T) {
		t.Parallel()
		data := []byte("raw data")
		b64 := base64.RawStdEncoding.EncodeToString(data)

		img, err := LoadImageFromBase64(b64, MediaTypeGIF, "anim.gif")
		require.NoError(t, err)
		require.Equal(t, data, img.Data)
	})

	t.Run("empty string returns ErrEmptyImageData", func(t *testing.T) {
		t.Parallel()
		_, err := LoadImageFromBase64("", MediaTypePNG, "test.png")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrEmptyImageData)
	})

	t.Run("invalid base64 returns error", func(t *testing.T) {
		t.Parallel()
		_, err := LoadImageFromBase64("!!!not-base64!!!", MediaTypePNG, "test.png")
		require.Error(t, err)
	})
}

func TestLoadImageFromURL(t *testing.T) {
	t.Parallel()

	t.Run("downloads image successfully", func(t *testing.T) {
		t.Parallel()
		imageData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imageData)
		}))
		defer server.Close()

		img, err := LoadImageFromURL(context.Background(), server.URL+"/screenshot.png")
		require.NoError(t, err)
		require.Equal(t, imageData, img.Data)
		require.Equal(t, MediaType("image/png"), img.MediaType)
		require.Equal(t, "screenshot.png", img.Filename)
	})

	t.Run("returns error on HTTP 404", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := LoadImageFromURL(context.Background(), server.URL+"/missing.png")
		require.Error(t, err)
		require.Contains(t, err.Error(), "404")
	})

	t.Run("returns error on invalid URL", func(t *testing.T) {
		t.Parallel()
		_, err := LoadImageFromURL(context.Background(), "http://[::1]:namedport")
		require.Error(t, err)
	})

	t.Run("rejects non-image response body", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("<html>not an image</html>"))
		}))
		defer server.Close()

		_, err := LoadImageFromURL(context.Background(), server.URL+"/fake.png")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidImage)
	})
}

func TestConfigValidationExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config with all params",
			config: Config{
				Model:            testModel(),
				Temperature:      0.5,
				TopP:             0.9,
				TopK:             40,
				PresencePenalty:  0.5,
				FrequencyPenalty: -0.5,
			},
			wantErr: nil,
		},
		{
			name:    "invalid top-p too high",
			config:  Config{Model: testModel(), TopP: 1.5},
			wantErr: ErrInvalidTopP,
		},
		{
			name:    "invalid top-p negative",
			config:  Config{Model: testModel(), TopP: -0.1},
			wantErr: ErrInvalidTopP,
		},
		{
			name:    "invalid top-k negative",
			config:  Config{Model: testModel(), TopK: -1},
			wantErr: ErrInvalidTopK,
		},
		{
			name:    "invalid presence penalty too high",
			config:  Config{Model: testModel(), PresencePenalty: 2.5},
			wantErr: ErrInvalidPresencePenalty,
		},
		{
			name:    "invalid frequency penalty too low",
			config:  Config{Model: testModel(), FrequencyPenalty: -2.5},
			wantErr: ErrInvalidFrequencyPenalty,
		},
		{
			name:    "boundary presence penalty 2.0",
			config:  Config{Model: testModel(), PresencePenalty: 2.0},
			wantErr: nil,
		},
		{
			name:    "boundary frequency penalty -2.0",
			config:  Config{Model: testModel(), FrequencyPenalty: -2.0},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConversation(t *testing.T) {
	t.Run("empty conversation has zero messages", func(t *testing.T) {
		conv := NewConversation()
		require.Equal(t, 0, conv.Len())
		require.Empty(t, conv.Messages())
	})

	t.Run("AddUserMessage appends user message", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("describe this", ImageSrc())

		require.Equal(t, 1, conv.Len())
		msgs := conv.Messages()
		require.Len(t, msgs, 1)
		require.NotEmpty(t, msgs[0].Content)
	})

	t.Run("AddAssistantMessage appends assistant message", func(t *testing.T) {
		conv := NewConversation()
		conv.AddAssistantMessage("here is the analysis")

		require.Equal(t, 1, conv.Len())
	})

	t.Run("multi-turn conversation accumulates messages", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("first question", ImageSrc())
		conv.AddAssistantMessage("first answer")
		conv.AddUserMessage("follow up", ImageSrc())

		require.Equal(t, 3, conv.Len())
	})

	t.Run("nil images filtered in AddUserMessage", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("test", nil, nil)

		require.Equal(t, 1, conv.Len())
		msgs := conv.Messages()
		require.Len(t, msgs[0].Content, 1)
	})

	t.Run("fluent chaining returns same conversation", func(t *testing.T) {
		conv := NewConversation()
		returned := conv.AddUserMessage("test")
		require.Same(t, conv, returned)
	})

	t.Run("Clear resets messages and returns same instance", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("first", ImageSrc())
		conv.AddAssistantMessage("answer")
		require.Equal(t, 2, conv.Len())

		returned := conv.Clear()

		require.Same(t, conv, returned)
		require.Equal(t, 0, conv.Len())
		require.Empty(t, conv.Messages())
	})
}

func TestAnalyzeConversation(t *testing.T) {
	t.Run("analyzes with conversation history", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		conv := NewConversation()
		conv.AddUserMessage("first question", ImageSrc())
		conv.AddAssistantMessage("first answer")

		result, err := agent.AnalyzeConversation(
			context.Background(),
			conv,
			"follow up",
			ImageSrc(),
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, mockResponseText, result.Text)
	})

	t.Run("returns error for empty prompt", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		conv := NewConversation()
		_, err = agent.AnalyzeConversation(context.Background(), conv, "", ImageSrc())
		require.ErrorIs(t, err, ErrEmptyPrompt)
	})

	t.Run("returns error for no images", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		conv := NewConversation()
		_, err = agent.AnalyzeConversation(context.Background(), conv, "test")
		require.ErrorIs(t, err, ErrNoImages)
	})
}

func TestAnalyzeConversationStream(t *testing.T) {
	t.Run("streams with conversation history", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		conv := NewConversation()
		conv.AddUserMessage("previous", ImageSrc())

		var chunks []string
		result, err := agent.AnalyzeConversationStream(
			context.Background(),
			conv,
			"describe",
			func(text string) error {
				chunks = append(chunks, text)
				return nil
			},
			ImageSrc(),
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, chunks)
	})
}

func TestBatchAnalysis(t *testing.T) {
	t.Run("analyzes multiple images concurrently", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		images := []*ImageSource{
			ImageSrc("a.png"),
			ImageSrc("b.png"),
			ImageSrc("c.png"),
		}

		results := agent.AnalyzeBatch(context.Background(), "describe", 2, images...)

		require.Len(t, results, 3)
		for i, r := range results {
			require.NoError(t, r.Err, "image %d should succeed", i)
			require.NotNil(t, r.Result)
			require.Equal(t, i, r.Index)
		}
	})

	t.Run("handles nil images gracefully", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		results := agent.AnalyzeBatch(
			context.Background(),
			"describe",
			1,
			ImageSrc("a.png"),
			nil,
			ImageSrc("c.png"),
		)

		require.Len(t, results, 3)
		require.NotNil(t, results[0].Result)
		require.Nil(t, results[1].Result)
		require.NotNil(t, results[2].Result)
	})

	t.Run("captures errors per-image", func(t *testing.T) {
		model := &mockModel{generateErr: errors.New("model down")}
		agent, err := NewAgent(Config{Model: model})
		require.NoError(t, err)

		results := agent.AnalyzeBatch(
			context.Background(),
			"describe",
			1,
			ImageSrc("a.png"),
		)

		require.Len(t, results, 1)
		require.Error(t, results[0].Err)
		require.Nil(t, results[0].Result)
	})
}

func TestHooks(t *testing.T) {
	t.Run("OnStart fires with prompt and image count", func(t *testing.T) {
		var mu sync.Mutex
		var gotPrompt string
		var gotCount int

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnStart: func(_ context.Context, prompt string, count int) {
					mu.Lock()
					defer mu.Unlock()
					gotPrompt = prompt
					gotCount = count
				},
			},
		})
		require.NoError(t, err)

		_, err = agent.Analyze(context.Background(), "test prompt", ImageSrc(), ImageSrc())
		require.NoError(t, err)

		mu.Lock()
		require.Equal(t, "test prompt", gotPrompt)
		require.Equal(t, 2, gotCount)
		mu.Unlock()
	})

	t.Run("OnFinish fires with result", func(t *testing.T) {
		var gotResult *AnalyzeResult

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnFinish: func(_ context.Context, result *AnalyzeResult) {
					gotResult = result
				},
			},
		})
		require.NoError(t, err)

		_, err = agent.Analyze(context.Background(), "test", ImageSrc())
		require.NoError(t, err)

		require.NotNil(t, gotResult)
		require.Equal(t, mockResponseText, gotResult.Text)
	})

	t.Run("OnError fires on model failure", func(t *testing.T) {
		var gotErr error

		agent, err := NewAgent(Config{
			Model: &mockModel{generateErr: errors.New("model exploded")},
			Hooks: Hooks{
				OnError: func(_ context.Context, e error) {
					gotErr = e
				},
			},
		})
		require.NoError(t, err)

		_, err = agent.Analyze(context.Background(), "test", ImageSrc())
		require.Error(t, err)

		require.Error(t, gotErr)
	})

	t.Run("hooks do not fire on validation errors", func(t *testing.T) {
		var fired atomic.Bool

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnStart: func(context.Context, string, int) { fired.Store(true) },
			},
		})
		require.NoError(t, err)

		_, err = agent.Analyze(context.Background(), "", ImageSrc())
		require.Error(t, err)
		require.False(t, fired.Load(), "OnStart should not fire on validation error")
	})
}

func TestHooksFireAcrossAllAnalysisMethods(t *testing.T) {
	t.Parallel()

	startTracker := func() (*sync.Mutex, *atomic.Int32) {
		return &sync.Mutex{}, &atomic.Int32{}
	}

	t.Run("AnalyzeConversation fires OnStart/OnFinish", func(t *testing.T) {
		t.Parallel()
		mu, starts := startTracker()
		var finished atomic.Bool

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnStart: func(_ context.Context, _ string, _ int) {
					mu.Lock()
					defer mu.Unlock()
					starts.Add(1)
				},
				OnFinish: func(_ context.Context, _ *AnalyzeResult) { finished.Store(true) },
			},
		})
		require.NoError(t, err)

		_, err = agent.AnalyzeConversation(context.Background(), NewConversation(), "prompt", ImageSrc())
		require.NoError(t, err)
		require.Equal(t, int32(1), starts.Load(), "OnStart must fire exactly once")
		require.True(t, finished.Load(), "OnFinish must fire")
	})

	t.Run("AnalyzeConversation fires OnError on model failure", func(t *testing.T) {
		t.Parallel()
		var gotErr atomic.Value

		agent, err := NewAgent(Config{
			Model: &mockModel{generateErr: errors.New("boom")},
			Hooks: Hooks{OnError: func(_ context.Context, _ error) { gotErr.Store(errors.New("fired")) }},
		})
		require.NoError(t, err)

		_, err = agent.AnalyzeConversation(context.Background(), NewConversation(), "prompt", ImageSrc())
		require.Error(t, err)
		require.NotNil(t, gotErr.Load(), "OnError must fire")
	})

	t.Run("AnalyzeConversationStream fires OnStart/OnFinish", func(t *testing.T) {
		t.Parallel()
		mu, starts := startTracker()
		var finished atomic.Bool

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnStart: func(_ context.Context, _ string, _ int) {
					mu.Lock()
					defer mu.Unlock()
					starts.Add(1)
				},
				OnFinish: func(_ context.Context, _ *AnalyzeResult) { finished.Store(true) },
			},
		})
		require.NoError(t, err)

		_, err = agent.AnalyzeConversationStream(
			context.Background(),
			NewConversation(),
			"prompt",
			func(string) error { return nil },
			ImageSrc(),
		)
		require.NoError(t, err)
		require.Equal(t, int32(1), starts.Load())
		require.True(t, finished.Load())
	})

	t.Run("AnalyzeStructured fires OnStart/OnFinish", func(t *testing.T) {
		t.Parallel()
		mu, starts := startTracker()
		var finished atomic.Bool

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnStart: func(_ context.Context, _ string, _ int) {
					mu.Lock()
					defer mu.Unlock()
					starts.Add(1)
				},
				OnFinish: func(_ context.Context, _ *AnalyzeResult) { finished.Store(true) },
			},
		})
		require.NoError(t, err)

		_, err = AnalyzeStructured[testReview](context.Background(), agent, "prompt", ImageSrc())
		require.NoError(t, err)
		require.Equal(t, int32(1), starts.Load())
		require.True(t, finished.Load())
	})

	t.Run("AnalyzeStructured fires OnError on model failure", func(t *testing.T) {
		t.Parallel()
		var gotErr atomic.Value

		agent, err := NewAgent(Config{
			Model: &mockModel{generateObjectErr: errors.New("structured boom")},
			Hooks: Hooks{OnError: func(_ context.Context, _ error) { gotErr.Store(errors.New("fired")) }},
		})
		require.NoError(t, err)

		_, err = AnalyzeStructured[testReview](context.Background(), agent, "prompt", ImageSrc())
		require.Error(t, err)
		require.NotNil(t, gotErr.Load(), "OnError must fire")
	})

	t.Run("AnalyzeStructuredStream fires OnStart/OnFinish", func(t *testing.T) {
		t.Parallel()
		mu, starts := startTracker()
		var finished atomic.Bool

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{
				OnStart: func(_ context.Context, _ string, _ int) {
					mu.Lock()
					defer mu.Unlock()
					starts.Add(1)
				},
				OnFinish: func(_ context.Context, _ *AnalyzeResult) { finished.Store(true) },
			},
		})
		require.NoError(t, err)

		_, err = AnalyzeStructuredStream[testReview](
			context.Background(),
			agent,
			"prompt",
			func(testReview) {},
			ImageSrc(),
		)
		require.NoError(t, err)
		require.Equal(t, int32(1), starts.Load())
		require.True(t, finished.Load())
	})

	t.Run("hooks do not fire on validation errors in wired methods", func(t *testing.T) {
		t.Parallel()
		var fired atomic.Bool

		agent, err := NewAgent(Config{
			Model: testModel(),
			Hooks: Hooks{OnStart: func(_ context.Context, _ string, _ int) { fired.Store(true) }},
		})
		require.NoError(t, err)

		_, err = agent.AnalyzeConversation(context.Background(), NewConversation(), "", ImageSrc())
		require.Error(t, err)
		require.False(t, fired.Load(), "OnStart must not fire on validation error")

		_, err = AnalyzeStructured[testReview](context.Background(), agent, "", ImageSrc())
		require.Error(t, err)
		require.False(t, fired.Load(), "OnStart must not fire on validation error")
	})
}

func TestAnalyzeStructuredStream(t *testing.T) {
	t.Run("streams partial objects via callback", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		var partials []testReview

		result, err := AnalyzeStructuredStream[testReview](
			context.Background(),
			agent,
			"analyze this",
			func(partial testReview) {
				partials = append(partials, partial)
			},
			ImageSrc(),
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, partials)
		require.Equal(t, testLayout, partials[0].Layout)
		require.Equal(t, testLayout, result.Object.Layout)
		require.Equal(t, int64(10), result.Usage.TotalTokens)
	})

	t.Run("returns error for empty prompt", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		_, err = AnalyzeStructuredStream[testReview](
			context.Background(),
			agent,
			"",
			nil,
			ImageSrc(),
		)
		require.ErrorIs(t, err, ErrEmptyPrompt)
	})

	t.Run("returns error for no images", func(t *testing.T) {
		agent, err := NewAgent(Config{Model: testModel()})
		require.NoError(t, err)

		_, err = AnalyzeStructuredStream[testReview](
			context.Background(),
			agent,
			"test",
			nil,
		)
		require.ErrorIs(t, err, ErrNoImages)
	})
}

func TestScreenshotAnalyzerCacheInvalidation(t *testing.T) {
	t.Run("config change after first analysis takes effect", func(t *testing.T) {
		sa := NewScreenshotAnalyzer(testModel())

		_, err := sa.AnalyzeScreenshotImages(context.Background(), "first", ImageSrc())
		require.NoError(t, err)

		require.NotNil(t, sa.cachedAgent, "agent should be cached after first call")

		sa.WithTemperature(0.9)

		require.Nil(t, sa.cachedAgent, "cache must be invalidated after config change")

		_, err = sa.AnalyzeScreenshotImages(context.Background(), "second", ImageSrc())
		require.NoError(t, err)

		require.NotNil(t, sa.cachedAgent)
		require.Equal(t, 0.9, sa.cachedAgent.config.Temperature)
	})

	t.Run("all builder methods invalidate cache", func(t *testing.T) {
		sa := NewScreenshotAnalyzer(testModel())

		// Force cache initialization
		_, err := sa.AnalyzeScreenshotImages(context.Background(), "init", ImageSrc())
		require.NoError(t, err)
		require.NotNil(t, sa.cachedAgent)

		methods := []func(*ScreenshotAnalyzer){
			func(s *ScreenshotAnalyzer) { s.WithSystemPrompt("new") },
			func(s *ScreenshotAnalyzer) { s.WithMaxOutputTokens(500) },
			func(s *ScreenshotAnalyzer) { s.WithTemperature(0.5) },
			func(s *ScreenshotAnalyzer) { s.WithTopP(0.8) },
			func(s *ScreenshotAnalyzer) { s.WithTopK(30) },
			func(s *ScreenshotAnalyzer) { s.WithPresencePenalty(0.3) },
			func(s *ScreenshotAnalyzer) { s.WithFrequencyPenalty(-0.3) },
			func(s *ScreenshotAnalyzer) { s.WithMaxRetries(5) },
			func(s *ScreenshotAnalyzer) { s.WithRequestTimeout(30e9) },
			func(s *ScreenshotAnalyzer) {
				s.WithHooks(Hooks{OnError: func(context.Context, error) {}})
			},
		}

		for i, m := range methods {
			m(sa)
			require.Nilf(t, sa.cachedAgent, "cache must be nil after builder method %d", i)
		}
	})
}
