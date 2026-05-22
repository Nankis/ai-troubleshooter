package worker

import (
	"context"
	"log"
	"sync"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/queue"
)

type CaseProcessor interface {
	ProcessCase(ctx context.Context, caseID int64) (caseflow.ProcessResult, error)
}

type Pool struct {
	queue       queue.Queue
	processor   CaseProcessor
	concurrency int
	wg          sync.WaitGroup
}

func NewPool(q queue.Queue, processor CaseProcessor, concurrency int) *Pool {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &Pool{queue: q, processor: processor, concurrency: concurrency}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.concurrency; i++ {
		workerID := i + 1
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.loop(ctx, workerID)
		}()
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) loop(ctx context.Context, workerID int) {
	for {
		event, err := p.queue.Consume(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker=%d consume_error=%v", workerID, err)
			continue
		}
		if event.CaseID == 0 {
			continue
		}
		result, err := p.processor.ProcessCase(ctx, event.CaseID)
		if err != nil {
			log.Printf("worker=%d case_id=%d process_error=%v", workerID, event.CaseID, err)
			continue
		}
		log.Printf("worker=%d case_no=%s status=%s reply=%q", workerID, result.CaseNo, result.Status, result.Reply)
	}
}
