//internal/decision/engine.go
package decision

import (
	"context"
	"sync"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/action"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/logger"
)

type Config struct {
	AggregationWindow time.Duration
}

type Engine struct {
	config       Config
	judge        *Judge
	actionEngine *action.Engine
	log          logger.Logger

	buffer map[string][]Signal
	timers map[string]*time.Timer
	mu     sync.Mutex

	inputCh chan Signal
}

func NewEngine(cfg Config, judge *Judge, actionEng *action.Engine, log logger.Logger) *Engine {
	return &Engine{
		config:       cfg,
		judge:        judge,
		actionEngine: actionEng,
		log:          log,
		buffer:       make(map[string][]Signal),
		timers:       make(map[string]*time.Timer),
		inputCh:      make(chan Signal, 1000),
	}
}

func (e *Engine) InputChannel() chan<- Signal {
	return e.inputCh
}

func (e *Engine) Start(ctx context.Context) {
	go e.loop(ctx)
}

func (e *Engine) loop(ctx context.Context) {
	for {
		select {
		case sig := <-e.inputCh:
			e.handleSignal(sig)
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) handleSignal(sig Signal) {
	e.mu.Lock()
	defer e.mu.Unlock()

	cid := sig.ContainerID
	e.buffer[cid] = append(e.buffer[cid], sig)

	if _, exists := e.timers[cid]; !exists {
		e.timers[cid] = time.AfterFunc(
			e.config.AggregationWindow,
			func() { e.flush(cid) },
		)
	}
}

func (e *Engine) flush(containerID string) {

	e.mu.Lock()
	signals := e.buffer[containerID]
	delete(e.buffer, containerID)
	delete(e.timers, containerID)
	e.mu.Unlock()

	if len(signals) == 0 {
		return
	}

	ctx := context.Background()

	incident, err := e.judge.Evaluate(ctx, containerID, signals)
	if err != nil {
		e.log.Error("Judge evaluate failed", err, map[string]interface{}{
			"container_id":  containerID,
			"signal_count": len(signals),
		})
		return
	}
	if incident == nil {
		return
	}

	e.dispatch(ctx, *incident)
}

func (e *Engine) dispatch(ctx context.Context, inc domain.Incident) {
	if e.actionEngine != nil {
		e.actionEngine.Dispatch(ctx, inc)
	}
}