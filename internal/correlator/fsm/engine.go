// internal/correlator/fsm/engine.go
package fsm

import (
	"context"
	"log"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type Engine struct {
	store *Store
}

func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

func (e *Engine) Process(
	ctx context.Context,
	containerID string,
	tactic string,
	chain *model.Chain,
) (*model.Alert, error) {

	// فقط برای زنجیره مورد نظر لاگ اضافه تولید کنیم که شلوغ نشود
	debugMode := (chain.ID == "remote_exploit_sample")

	ttl, err := time.ParseDuration(chain.MaxDuration)
	if err != nil {
		return nil, err
	}

	state, err := e.store.Get(ctx, containerID, chain.ID)
	if err != nil {
		log.Printf("❌ REDIS ERROR: %v", err)
		return nil, err
	}

	// 1. حالت شروع جدید
	if state == nil {
		if tactic == chain.Sequence[0] {
			if err := e.store.Create(ctx, containerID, chain.ID, tactic, ttl); err != nil {
				return nil, err
			}
			log.Printf("🟢 FSM START | Chain=%s | Tactic=%s", chain.ID, tactic)
		} else {
			// اگر دیباگ مود بود و استیت نداشتیم، بگو چرا رد کرد
			if debugMode {
				log.Printf("👻 IGNORED (No State) | Chain=%s | Got=%s | ExpectedStart=%s", 
					chain.ID, tactic, chain.Sequence[0])
			}
		}
		return nil, nil
	}

	// 2. حالت ریست (فقط اگر وسط زنجیره بودیم)
	if tactic == chain.Sequence[0] && state.Step > 0 {
		log.Printf("🔄 FSM RESET | Chain=%s | Container=%s | Restarting sequence", chain.ID, containerID)
		if err := e.store.Create(ctx, containerID, chain.ID, tactic, ttl); err != nil {
			return nil, err
		}
		return nil, nil
	}
	
	// 3. حالت تکراری (Keep Alive)
	if tactic == chain.Sequence[0] && state.Step > 0 {
		if debugMode {
			log.Printf("♻️ IGNORED (Duplicate) | Chain=%s | Step=%d | Tactic=%s", chain.ID, state.Step, tactic)
		}
		return nil, nil
	}

	// 4. حالت پیشرفت (Promote)
	nextStep := state.Step + 1
	if nextStep >= len(chain.Sequence) {
		return nil, nil
	}

	expectedTactic := chain.Sequence[nextStep]

	// *** لاگ حیاتی برای پیدا کردن مشکل ***
	if tactic != expectedTactic {
		if debugMode {
			log.Printf("⛔ MISMATCH | Chain=%s | CurrentStep=%d | Expected=%q | Got=%q", 
				chain.ID, state.Step, expectedTactic, tactic)
		}
		return nil, nil
	}

	// اگر مچ شد:
	if nextStep == len(chain.Sequence)-1 {
		// تکمیل زنجیره
		timeline, _ := e.store.GetTimeline(ctx, containerID, chain.ID)
		alert := BuildAlert(containerID, chain, state, timeline)
		log.Printf("CHAIN COMPLETED | Chain=%s | Severity=%s", chain.ID, alert.Severity)
		_ = e.store.Delete(ctx, containerID, chain.ID)
		return alert, nil
	}

	if err := e.store.Promote(ctx, containerID, chain.ID, tactic, nextStep, ttl); err != nil {
		return nil, err
	}

	log.Printf("🔼 FSM PROMOTE | Chain=%s | Step=%d | Tactic=%s", chain.ID, nextStep, tactic)
	return nil, nil
}
