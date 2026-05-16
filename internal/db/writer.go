package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// ErrWriterClosed is returned when a write is submitted after the writer
// has been stopped.
var ErrWriterClosed = errors.New("db writer closed")

// WriteFn performs the actual write inside a transaction. Implementors
// must use the supplied *sql.Tx exclusively; do not capture the parent
// *sql.DB.
type WriteFn func(tx *sql.Tx) error

// writeJob is a single unit of work for the writer goroutine.
type writeJob struct {
	ctx context.Context
	fn  WriteFn
	res chan error
}

// Writer serialises every mutation against the SQLite file via a single
// goroutine fed by a channel. modernc.org/sqlite is single-writer; we
// avoid SQLITE_BUSY by funnelling all writes here. Reads remain free.
type Writer struct {
	sqlDB *sql.DB

	jobs   chan writeJob
	stopCh chan struct{}
	doneCh chan struct{}
	once   sync.Once
}

func newWriter(sqlDB *sql.DB) *Writer {
	return &Writer{
		sqlDB:  sqlDB,
		jobs:   make(chan writeJob, 64),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (w *Writer) start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *Writer) loop(parent context.Context) {
	defer close(w.doneCh)
	for {
		select {
		case <-parent.Done():
			w.drain(parent.Err())
			return
		case <-w.stopCh:
			w.drain(ErrWriterClosed)
			return
		case j, ok := <-w.jobs:
			if !ok {
				return
			}
			j.res <- w.run(j)
		}
	}
}

func (w *Writer) run(j writeJob) error {
	tx, err := w.sqlDB.BeginTx(j.ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := j.fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// drain rejects all queued jobs with err. New submissions also fail
// because stop() closes jobs.
func (w *Writer) drain(err error) {
	for {
		select {
		case j, ok := <-w.jobs:
			if !ok {
				return
			}
			j.res <- err
		default:
			return
		}
	}
}

// Submit enqueues fn to run inside a serialised transaction and waits
// for the result. Safe for concurrent callers.
func (w *Writer) Submit(ctx context.Context, fn WriteFn) error {
	res := make(chan error, 1)
	job := writeJob{ctx: ctx, fn: fn, res: res}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stopCh:
		return ErrWriterClosed
	case w.jobs <- job:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-res:
		return err
	}
}

// stop halts the writer goroutine. Idempotent.
func (w *Writer) stop() {
	w.once.Do(func() {
		close(w.stopCh)
	})
	<-w.doneCh
}
