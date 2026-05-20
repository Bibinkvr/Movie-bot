package core

import (
	"context"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// Job represents a unit of work to be processed by a worker.
type Job struct {
	Bot     *gotgbot.Bot
	Ctx     *ext.Context
	Handler func(bot *gotgbot.Bot, ctx *ext.Context) error
}

type WorkerPool struct {
	JobQueue chan Job
	Workers  int
}

func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		JobQueue: make(chan Job, queueSize),
		Workers:  workers,
	}
}

func (p *WorkerPool) Start(ctx context.Context, log *zap.Logger) {
	log.Info("starting worker pool", zap.Int("workers", p.Workers))
	for i := 0; i < p.Workers; i++ {
		go func(id int) {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-p.JobQueue:
					err := job.Handler(job.Bot, job.Ctx)
					if err != nil {
						log.Warn("worker: handler failed", zap.Int("worker_id", id), zap.Error(err))
					}
				}
			}
		}(i)
	}
}

func (p *WorkerPool) Push(job Job) bool {
	select {
	case p.JobQueue <- job:
		return true
	default:
		// Queue is full, drop job
		return false
	}
}
