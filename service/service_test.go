package service

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/bot"
	"github.com/mytelegrambot/deepseek"
	"github.com/mytelegrambot/storage"
	"reflect"
	"testing"
)

func TestNewService(t *testing.T) {
	type args struct {
		storage storage.Storage
		r1      deepseek.R1
		b       bot.BotAPI
	}
	tests := []struct {
		name string
		args args
		want *Service
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewService(tt.args.storage, tt.args.r1, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_ListCommands(t *testing.T) {
	type fields struct {
		storage storage.Storage
		r1      deepseek.R1
		bot     bot.BotAPI
	}
	type args struct {
		ctx context.Context
		msg *tgbotapi.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
				bot:     tt.fields.bot,
			}
			got, err := s.ListCommands(tt.args.ctx, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListCommands() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_ProcessMessage(t *testing.T) {
	type fields struct {
		storage storage.Storage
		r1      deepseek.R1
		bot     bot.BotAPI
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
			s := &Service{
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
				bot:     tt.fields.bot,
			}
			if err := s.ProcessMessage(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("ProcessMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_SetBot(t *testing.T) {
	type fields struct {
		storage storage.Storage
		r1      deepseek.R1
		bot     bot.BotAPI
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
			s := &Service{
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
				bot:     tt.fields.bot,
			}
			if err := s.SetBot(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("SetBot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_getAiResponse(t *testing.T) {
	type fields struct {
		storage storage.Storage
		r1      deepseek.R1
		bot     bot.BotAPI
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
			s := &Service{
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
				bot:     tt.fields.bot,
			}
			if err := s.getAiResponse(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("getAiResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_processCommand(t *testing.T) {
	type fields struct {
		storage storage.Storage
		r1      deepseek.R1
		bot     bot.BotAPI
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
			s := &Service{
				storage: tt.fields.storage,
				r1:      tt.fields.r1,
				bot:     tt.fields.bot,
			}
			if err := s.processCommand(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("processCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
