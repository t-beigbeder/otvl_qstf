package common

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJobController(t *testing.T) {
	logger := GetLoggerFor("TestJobController")
	getJob := func(ctx context.Context, label string, err error) *Job {
		logger := logger.With("doer", label)
		job := &Job{
			Label: label,
			Run: func(ctx context.Context, arg any) error {
				logger.Info("doing", "arg", arg)
				select {
				case <-ctx.Done():
					logger.Info("context done")
					return err
				}
			},
		}
		return job
	}
	jc := NewJobController(logger.With("controller", "this"))
	ctx, cancel := context.WithCancel(context.Background())
	err := jc.RunJob(ctx, getJob(context.Background(), "job1", nil), "arg1")
	require.NoError(t, err)
	err = jc.RunJob(ctx, getJob(context.Background(), "job1", nil), nil)
	require.NotNil(t, err)
	err = jc.RunJob(ctx, getJob(context.Background(), "job2", errors.New("error on job2")), nil)
	require.NoError(t, err)
	go func() {
		cancel()
	}()
	jc.Shutdown()
}
