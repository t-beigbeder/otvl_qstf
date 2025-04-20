package common

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type jobctl struct {
	mx      sync.Mutex
	wg      sync.WaitGroup
	running map[string]*Job
	logger  *slog.Logger
}

type Job struct {
	Label string
	Run   func(context.Context, any) error
}

type JobController interface {
	RunJob(ctx context.Context, job *Job, arg any) error
	Shutdown()
}

func NewJobController(logger *slog.Logger) JobController {
	jc := &jobctl{
		running: make(map[string]*Job),
		logger:  logger,
	}
	return jc
}

func (jc *jobctl) RunJob(ctx context.Context, job *Job, arg any) error {
	jc.mx.Lock()
	defer jc.mx.Unlock()
	_, ok := jc.running[job.Label]
	if ok {
		return fmt.Errorf("runJob %s already running", job.Label)
	}
	jc.running[job.Label] = job
	jc.logger.Info("runJob job start", "job", job.Label)
	jc.wg.Add(1)

	go func() {
		defer jc.wg.Done()
		err := job.Run(ctx, arg)
		if err == nil {
			jc.logger.Info("runJob job done", "job", job.Label)
		} else {
			jc.logger.Error("runJob job error", "job", job.Label, "err", err.Error())
		}
		jc.mx.Lock()
		defer jc.mx.Unlock()
		delete(jc.running, job.Label)
	}()
	return nil
}

func (jc *jobctl) Shutdown() {
	jc.logger.Info("shutdown started", "running", len(jc.running))
	jc.wg.Wait()
	jc.logger.Info("shutdown done")
}
