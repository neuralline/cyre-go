// state/state.go
// Clean StateManager - The Big State Object (like main app state in JS)

package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/neuralline/cyre-go/types"
)

/*
	StateManager - The Big State Container

	Like a main JavaScript app state object:
	{
		channels: {},   // ioStore
		handlers: {},   // subscribers
		data: {},       // payloads
		timers: {},     // timeline
		workflows: {}   // branches
	}

	Pure storage coordinator - no intelligence, just data management
*/

// === GLOBAL SINGLETON ===

var (
	GlobalState *StateManager
	initOnce    sync.Once
)

// StateManager is the main state container
type StateManager struct {
	// Core stores (like JS object properties)
	ioStore     *Store[*types.IO]    // channels: {}
	subscribers *Store[*Subscriber]  // handlers: {}
	payloads    *Store[*Payload]     // data: {}
	timeline    *Store[*Timer]       // timers: {}
	branches    *Store[*BranchStore] // workflows: {}

	// Protection state (simple maps)
	throttleMap sync.Map
	debounceMap sync.Map

	// MetricState reference (for notifications only)
	metricState *MetricState
}

// === INITIALIZATION ===

// GetStateManager returns the global state manager
func GetStateManager() *StateManager {
	initOnce.Do(func() {
		GlobalState = &StateManager{
			ioStore:     NewStore[*types.IO](),
			subscribers: NewStore[*Subscriber](),
			payloads:    NewStore[*Payload](),
			timeline:    NewStore[*Timer](),
			branches:    NewStore[*BranchStore](),
			metricState: GetMetricState(),
		}
	})
	return GlobalState
}

// === DIRECT STORE ACCESS (No Wrappers) ===

// IO returns direct access to IO store (channels)
func IO() *Store[*types.IO] {
	return GetStateManager().ioStore
}

// Subscribers returns direct access to subscriber store (handlers)
func Subscribers() *Store[*Subscriber] {
	return GetStateManager().subscribers
}

// Timeline returns direct access to timeline store (timers)
func Timeline() *Store[*Timer] {
	return GetStateManager().timeline
}

// Branches returns direct access to branch store (workflows)
func Branches() *Store[*BranchStore] {
	return GetStateManager().branches
}

// === PAYLOAD MANAGEMENT (Special Case - Needs Conversion) ===

// SetPayload stores payload for an action
func SetPayload(actionID string, payload interface{}) {
	sm := GetStateManager()
	payloadObj := &Payload{
		ID:      actionID,
		Payload: payload,
	}
	sm.payloads.Set(actionID, payloadObj)
}

// GetPayload retrieves payload for an action
func GetPayload(actionID string) (interface{}, bool) {
	sm := GetStateManager()
	if payloadObj, exists := sm.payloads.Get(actionID); exists {
		return payloadObj.Payload, true
	}
	return nil, false
}

// ForgetPayload removes payload for an action
func ForgetPayload(actionID string) bool {
	sm := GetStateManager()
	return sm.payloads.Forget(actionID)
}

// ComparePayload compares current payload with new payload
func ComparePayload(actionID string, newPayload interface{}) bool {
	currentPayload, exists := GetPayload(actionID)
	if !exists {
		return true // New payload
	}

	if newPayload == nil && currentPayload == nil {
		return false
	}
	if newPayload == nil || currentPayload == nil {
		return true
	}

	currentJSON, err1 := json.Marshal(currentPayload)
	newJSON, err2 := json.Marshal(newPayload)

	if err1 != nil || err2 != nil {
		return true
	}

	return string(currentJSON) != string(newJSON)
}

// === PROTECTION STATE (Simple Storage) ===

func SetThrottleTime(actionID string, execTime time.Time) {
	GetStateManager().throttleMap.Store(actionID, execTime)
}

func GetThrottleTime(actionID string) (time.Time, bool) {
	sm := GetStateManager()
	if value, exists := sm.throttleMap.Load(actionID); exists {
		return value.(time.Time), true
	}
	return time.Time{}, false
}

func SetDebounceTimer(actionID string, timer interface{}) {
	GetStateManager().debounceMap.Store(actionID, timer)
}

func GetDebounceTimer(actionID string) (interface{}, bool) {
	sm := GetStateManager()
	if value, exists := sm.debounceMap.Load(actionID); exists {
		return value, true
	}
	return nil, false
}

func ClearDebounceTimer(actionID string) {
	GetStateManager().debounceMap.Delete(actionID)
}

// === METRIC STATE NOTIFICATION (Observer Pattern) ===

// notifyMetricState sends store count updates to MetricState
func notifyMetricState() {
	sm := GetStateManager()
	if sm.metricState == nil {
		return
	}

	// Simple count extraction
	channels := int(sm.ioStore.Count())
	subscribers := int(sm.subscribers.Count())
	timeline := int(sm.timeline.Count())
	branches := int(sm.branches.Count())
	tasks := timeline // Use timeline count as tasks

	// Notify MetricState
	sm.metricState.UpdateStoreCounts(channels, branches, tasks, subscribers, timeline)
}

// === STORE OPERATIONS WITH NOTIFICATIONS ===

// ActionSet stores an action and notifies metrics
func ActionSet(action *types.IO) error {
	if action.ID == "" {
		return fmt.Errorf("action must have an ID")
	}

	action.Timestamp = time.Now().UnixMilli()
	if action.Type == "" {
		action.Type = action.ID
	}

	IO().Set(action.ID, action)
	notifyMetricState()
	return nil
}

// ActionForget removes an action and notifies metrics
func ActionForget(actionID string) bool {
	ForgetPayload(actionID)
	result := IO().Forget(actionID)
	notifyMetricState()
	return result
}

// SubscriberAdd adds a subscriber and notifies metrics
func SubscriberAdd(subscriber *Subscriber) error {
	if subscriber.ID == "" || subscriber.Handler == nil {
		return fmt.Errorf("invalid subscriber format")
	}

	Subscribers().Set(subscriber.ID, subscriber)
	notifyMetricState()
	return nil
}

// SubscriberForget removes a subscriber and notifies metrics
func SubscriberForget(actionID string) bool {
	result := Subscribers().Forget(actionID)
	notifyMetricState()
	return result
}

// TimelineAdd adds a timer and notifies metrics
func TimelineAdd(timer *Timer) error {
	if timer.ID == "" {
		return fmt.Errorf("timer must have an ID")
	}

	Timeline().Set(timer.ID, timer)
	notifyMetricState()

	// Update active formation count
	sm := GetStateManager()
	if sm.metricState != nil {
		activeCount := len(TimelineGetActive())
		sm.metricState.SetActiveFormations(activeCount)
	}

	return nil
}

// TimelineForget removes a timer and notifies metrics
func TimelineForget(timerID string) bool {
	if timer, exists := Timeline().Get(timerID); exists {
		timer.Status = "cancelled"
	}

	result := Timeline().Forget(timerID)
	notifyMetricState()

	// Update active formation count
	sm := GetStateManager()
	if sm.metricState != nil {
		activeCount := len(TimelineGetActive())
		sm.metricState.SetActiveFormations(activeCount)
	}

	return result
}

// TimelineGetActive returns active timers
func TimelineGetActive() []*Timer {
	timers := Timeline().GetAll()
	var active []*Timer
	for _, timer := range timers {
		if timer.Status == "active" {
			active = append(active, timer)
		}
	}
	return active
}

// === CLEANUP OPERATIONS ===

// Clear clears all state
func Clear() {
	sm := GetStateManager()
	sm.ioStore.Clear()
	sm.subscribers.Clear()
	sm.payloads.Clear()
	sm.timeline.Clear()
	sm.branches.Clear()

	// Clear protection maps
	sm.throttleMap.Range(func(key, value interface{}) bool {
		sm.throttleMap.Delete(key)
		return true
	})
	sm.debounceMap.Range(func(key, value interface{}) bool {
		sm.debounceMap.Delete(key)
		return true
	})

	notifyMetricState()
}
