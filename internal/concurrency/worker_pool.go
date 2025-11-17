package concurrency

import (
	"context"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// EventProcessor обрабатывает события
type EventProcessor interface {
	Process(ctx context.Context, event domain.Event) (*domain.BotResponse, error)
}

// WorkerPool обрабатывает события конкурентно через пул воркеров
type WorkerPool struct {
	workers      int
	eventChan    chan domain.Event
	processor    EventProcessor
	logger       *zap.SugaredLogger
	responseChan chan *Response
}

// Response представляет результат обработки события
type Response struct {
	Event    domain.Event
	Response *domain.BotResponse
	Error    error
}

// NewWorkerPool создает новый WorkerPool
func NewWorkerPool(workers int, processor EventProcessor, logger *zap.SugaredLogger) *WorkerPool {
	return &WorkerPool{
		workers:      workers,
		eventChan:    make(chan domain.Event, 100),
		processor:    processor,
		logger:       logger.Named("worker_pool"),
		responseChan: make(chan *Response, 100),
	}
}

// Submit добавляет событие в очередь обработки
func (wp *WorkerPool) Submit(event domain.Event) {
	wp.eventChan <- event
	wp.logger.Debugw("Event submitted to worker pool",
		"event_type", event.Type(),
		"chat_id", event.ChatID(),
		"user_id", event.UserID(),
		"queue_length", len(wp.eventChan),
	)
}

// Start запускает пул воркеров
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		go wp.worker(ctx, i)
	}
	wp.logger.Infof("Started %d workers", wp.workers)
}

// worker обрабатывает события из канала
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	wp.logger.Debugf("Worker %d started", id)
	defer wp.logger.Debugf("Worker %d stopped", id)

	for {
		select {
		case event := <-wp.eventChan:
			if event == nil {
				wp.logger.Warnf("Worker %d received nil event", id)
				continue
			}
			wp.logger.Debugw("Worker processing event",
				"worker_id", id,
				"event_type", event.Type(),
				"chat_id", event.ChatID(),
				"user_id", event.UserID(),
			)
			response, err := wp.processor.Process(ctx, event)
			if err != nil {
				wp.logger.Errorw("Worker failed to process event",
					"worker_id", id,
					"event_type", event.Type(),
					"chat_id", event.ChatID(),
					"user_id", event.UserID(),
					"error", err,
				)
			} else {
				wp.logger.Debugw("Worker processed event successfully",
					"worker_id", id,
					"event_type", event.Type(),
					"chat_id", event.ChatID(),
					"user_id", event.UserID(),
				)
			}
			wp.responseChan <- &Response{
				Event:    event,
				Response: response,
				Error:    err,
			}
		case <-ctx.Done():
			wp.logger.Debugf("Worker %d received shutdown signal", id)
			return
		}
	}
}

// Responses возвращает канал с результатами обработки
func (wp *WorkerPool) Responses() <-chan *Response {
	return wp.responseChan
}

// Stop останавливает пул воркеров
func (wp *WorkerPool) Stop() {
	close(wp.eventChan)
	close(wp.responseChan)
	wp.logger.Info("Worker pool stopped")
}
