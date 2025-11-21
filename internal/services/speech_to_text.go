package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/vlks-dev/mytelegrambotapi/bot"
	"github.com/vlks-dev/mytelegrambotapi/config"
	"go.uber.org/zap"
)

// SpeechToTextService интерфейс для преобразования речи в текст
type SpeechToTextService interface {
	// Transcribe преобразует голосовое сообщение в текст
	Transcribe(ctx context.Context, fileID string) (string, error)
}

// whisperHTTPClient реализация SpeechToTextService через Whisper HTTP сервис
type whisperHTTPClient struct {
	botAPI            bot.ExtendedBotAPI
	whisperServiceURL string
	tempDir           string
	httpClient        *http.Client
	logger            *zap.SugaredLogger
}

// NewWhisperHTTPClient создает новый Whisper HTTP клиент
func NewWhisperHTTPClient(botAPI bot.ExtendedBotAPI, cfg *config.Config, logger *zap.SugaredLogger) SpeechToTextService {
	if cfg.WhisperServiceURL == "" {
		logger.Warn("Whisper service URL not configured, using stub")
		return NewSpeechToTextServiceStub()
	}

	return &whisperHTTPClient{
		botAPI:            botAPI,
		whisperServiceURL: cfg.WhisperServiceURL,
		tempDir:           cfg.TempDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.Named("whisper_client"),
	}
}

// Transcribe преобразует голосовое сообщение в текст
func (s *whisperHTTPClient) Transcribe(ctx context.Context, fileID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	s.logger.Debugw("Starting transcription",
		"file_id", fileID,
	)

	// Шаг 2: Скачиваем файл напрямую по fileID
	originalPath := filepath.Join(s.tempDir, fmt.Sprintf("voice_%s_%d.ogg", fileID, time.Now().Unix()))
	err := s.botAPI.DownloadFile(ctx, fileID, originalPath)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer os.Remove(originalPath)

	s.logger.Debugw("File downloaded",
		"file_id", fileID,
		"path", originalPath,
	)

	// Шаг 3: Конвертируем в WAV (если нужно)
	wavPath := filepath.Join(s.tempDir, fmt.Sprintf("voice_%s_%d.wav", fileID, time.Now().Unix()))
	err = s.convertToWAV(ctx, originalPath, wavPath)
	if err != nil {
		s.logger.Warnw("Failed to convert to WAV, trying original file",
			"error", err,
		)
		// Пробуем отправить оригинальный файл
		wavPath = originalPath
	} else {
		defer os.Remove(wavPath)
	}

	// Шаг 4: Отправляем в Whisper сервис с retry
	var transcription string
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Infow("Retrying transcription",
				"attempt", attempt+1,
				"file_id", fileID,
			)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		transcription, err = s.sendToWhisper(ctx, wavPath)
		if err == nil {
			break
		}

		if ctx.Err() != nil {
			return "", fmt.Errorf("transcription attempt timeout: %w", ctx.Err())
		}

		s.logger.Warnw("Transcription attempt failed",
			"attempt", attempt+1,
			"error", err,
		)
	}

	if err != nil {
		return "", fmt.Errorf("transcription failed after %d attempts: %w", maxRetries+1, err)
	}

	s.logger.Debugw("Transcription completed",
		"file_id", fileID,
		"text_length", len(transcription),
	)

	return transcription, nil
}

// convertToWAV конвертирует аудио файл в WAV формат используя ffmpeg
func (s *whisperHTTPClient) convertToWAV(ctx context.Context, inputPath, outputPath string) error {
	// Проверяем, что ffmpeg доступен
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}

	// Конвертируем: OGG/OPUS -> WAV (16kHz, моно)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-ar", "16000", // Sample rate 16kHz
		"-ac", "1", // Mono channel
		"-y", // Overwrite output file
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Debugw("Audio converted to WAV",
		"input", inputPath,
		"output", outputPath,
	)
	return nil
}

// sendToWhisper отправляет файл в Whisper сервис
func (s *whisperHTTPClient) sendToWhisper(ctx context.Context, filePath string) (string, error) {
	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Создаем multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Добавляем файл
	part, err := writer.CreateFormFile("audio", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("copy file to form: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "POST", s.whisperServiceURL+"/transcribe", &requestBody)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Выполняем запрос
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper service error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var response struct {
		Text string `json:"text"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if response.Text == "" {
		return "", fmt.Errorf("empty transcription result")
	}

	return response.Text, nil
}

// speechToTextServiceStub заглушка реализации SpeechToTextService
type speechToTextServiceStub struct{}

// NewSpeechToTextServiceStub создает заглушку SpeechToTextService
func NewSpeechToTextServiceStub() SpeechToTextService {
	return &speechToTextServiceStub{}
}

// Transcribe заглушка метода преобразования речи в текст
func (s *speechToTextServiceStub) Transcribe(ctx context.Context, fileID string) (string, error) {
	return "заглушка для голосового сообщения", nil
}
