package bot

import (
	"context"
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/models"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func Test_parseChoices(t *testing.T) {
	type args struct {
		data string
	}
	// валидный ответ с choices
	response, _ := json.Marshal(models.CompletionResponse{
		ID:       "123",
		Provider: "test-ai",
		Model:    "test-model",
		Choices: []models.Choice{
			{
				Message: models.R1Message{
					Role:    "assistant",
					Content: "Ответ",
				},
			},
		},
		Usage: models.Usage{
			TotalTokens: 15,
		},
	})

	// ответ без choices, но с ошибкой
	noTokenErr, _ := json.Marshal(models.ErrorR1Message{
		Message: "Rate limit exceeded: free-models-per-day. Add 10 credits to unlock 1000 free model requests per day",
		Code:    429,
	})

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "valid completion response",
			args:    args{data: string(response)},
			want:    []string{"Ответ", "потрачено 15 токенов"},
			wantErr: false,
		},
		{
			name:    "valid error response",
			args:    args{data: string(noTokenErr)},
			want:    []string{"Закончились токены! Попробуйте завтра"},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			args:    args{data: "{"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChoices(tt.args.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBot_GetUpdates(t *testing.T) {
	type fields struct {
		api     *tgbotapi.BotAPI
		storage Storage
		r1      R1Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bot{
				api:     tt.fields.api,
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
			}
			if err := b.GetUpdates(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("GetUpdates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBot_processIncoming(t *testing.T) {
	type fields struct {
		api     *tgbotapi.BotAPI
		storage Storage
		r1      R1Client
	}
	type args struct {
		ctx context.Context
		msg *tgbotapi.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bot{
				api:     tt.fields.api,
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
			}
			if err := b.processIncoming(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("processIncoming() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBot_saveMessage(t *testing.T) {
	type fields struct {
		api     *tgbotapi.BotAPI
		storage Storage
		r1      R1Client
	}
	type args struct {
		ctx context.Context
		m   *models.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bot{
				api:     tt.fields.api,
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
			}
			if err := b.saveMessage(tt.args.ctx, tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("saveMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBot_sendAnswer(t *testing.T) {
	type fields struct {
		api     *tgbotapi.BotAPI
		storage Storage
		r1      R1Client
	}
	type args struct {
		chatID int64
		text   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *tgbotapi.Message
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bot{
				api:     tt.fields.api,
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
			}
			got, err := b.sendAnswer(tt.args.chatID, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("sendAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sendAnswer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBot(t *testing.T) {
	type args struct {
		config  *config.Config
		storage Storage
		r1      R1Client
	}
	tests := []struct {
		name    string
		args    args
		want    *Bot
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBot(tt.args.config, tt.args.storage, tt.args.r1)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBot() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseChoices1(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChoices(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseChoices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseChoices() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_truncate(t *testing.T) {
	type args struct {
		s        string
		maxRunes int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.args.s, tt.args.maxRunes); got != tt.want {
				t.Errorf("truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}
