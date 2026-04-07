package swarm

import (
	"log/slog"
	"math"
	"sync"
)

// CapacityClass represents infrastructure tiers usable for burst capacity.
type CapacityClass string

const (
	CapacityCloud       CapacityClass = "cloud"
	CapacityEdge        CapacityClass = "edge"
	CapacityResidential CapacityClass = "residential"
)

// ScaleAction describes an adjustment to a capacity class.
type ScaleAction struct {
	Class  CapacityClass
	Delta  int
	Reason string
}

// ElasticityManager recommends scale actions so capacity follows demand curves.
type ElasticityManager struct {
	logger     *slog.Logger
	mu         sync.Mutex
	capacity   map[CapacityClass]int
	minPerTier int
	targetUtil float64
}

// NewElasticityManager creates a manager with initial capacities.
func NewElasticityManager(logger *slog.Logger, initial map[CapacityClass]int, targetUtil float64) *ElasticityManager {
	if targetUtil <= 0 {
		targetUtil = 0.7
	}
	cap := map[CapacityClass]int{
		CapacityCloud:       0,
		CapacityEdge:        0,
		CapacityResidential: 0,
	}
	for k, v := range initial {
		cap[k] = v
	}
	return &ElasticityManager{logger: logger, capacity: cap, minPerTier: 0, targetUtil: targetUtil}
}

// ObserveDemand computes scale actions to align capacity with demand at the target utilization.
func (em *ElasticityManager) ObserveDemand(demandUnits int) []ScaleAction {
	em.mu.Lock()
	defer em.mu.Unlock()

	current := em.totalCapacity()
	if current == 0 {
		current = 1
	}
	if demandUnits < 0 {
		demandUnits = 0
	}

	required := int(math.Ceil(float64(demandUnits) / em.targetUtil))
	if required < em.minTotal() {
		required = em.minTotal()
	}

	actions := []ScaleAction{}
	if required > current {
		delta := required - current
		actions = append(actions, ScaleAction{Class: CapacityCloud, Delta: delta, Reason: "scale-up"})
	} else {
		delta := current - required
		if delta > 0 {
			actions = append(actions, em.scaleDown(delta)...)
		}
	}

	return actions
}

// Apply mutates internal capacities according to the provided actions.
func (em *ElasticityManager) Apply(actions []ScaleAction) {
	em.mu.Lock()
	defer em.mu.Unlock()
	for _, act := range actions {
		em.capacity[act.Class] += act.Delta
		if em.capacity[act.Class] < em.minPerTier {
			em.capacity[act.Class] = em.minPerTier
		}
	}
}

// Snapshot returns a copy of current capacities.
func (em *ElasticityManager) Snapshot() map[CapacityClass]int {
	em.mu.Lock()
	defer em.mu.Unlock()
	out := make(map[CapacityClass]int, len(em.capacity))
	for k, v := range em.capacity {
		out[k] = v
	}
	return out
}

func (em *ElasticityManager) totalCapacity() int {
	sum := 0
	for _, v := range em.capacity {
		sum += v
	}
	return sum
}

func (em *ElasticityManager) minTotal() int {
	return em.minPerTier * len(em.capacity)
}

func (em *ElasticityManager) scaleDown(delta int) []ScaleAction {
	actions := []ScaleAction{}
	classes := []CapacityClass{CapacityCloud, CapacityEdge, CapacityResidential}
	for _, cls := range classes {
		if delta == 0 {
			break
		}
		available := em.capacity[cls] - em.minPerTier
		if available <= 0 {
			continue
		}
		use := delta
		if use > available {
			use = available
		}
		if use > 0 {
			actions = append(actions, ScaleAction{Class: cls, Delta: -use, Reason: "scale-down"})
			delta -= use
		}
	}
	return actions
}
