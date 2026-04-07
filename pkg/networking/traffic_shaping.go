package networking

import (
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// DiurnalCurve captures per-hour multipliers to mimic human-like traffic ebbs/flows.
type DiurnalCurve struct {
	// Factors must be 24-length, indexed by hour 0-23. Values <=0 fall back to 1.0.
	Factors [24]float64
}

// Factor returns the multiplier for the provided time.
func (d DiurnalCurve) Factor(ts time.Time) float64 {
	hour := ts.Hour() % 24
	f := d.Factors[hour]
	if f <= 0 {
		return 1.0
	}
	return f
}

// TrafficShaper enforces per-peer bandwidth caps that mimic baseline network statistics.
// It applies diurnal curves and geographic normalization to blend into expected traffic patterns.
type TrafficShaper struct {
	mu sync.Mutex

	baseBytesPerMinute int
	diurnal            DiurnalCurve
	geoFactors         map[string]float64
	baselineMbps       float64

	peerBuckets map[peer.ID]*bucket
}

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// NewTrafficShaper builds a shaper with defaults targeting residential-looking traffic.
func NewTrafficShaper(baseBytesPerMinute int, diurnal DiurnalCurve, geo map[string]float64, baselineMbps float64) *TrafficShaper {
	if baseBytesPerMinute <= 0 {
		baseBytesPerMinute = 5 * 1024 * 1024 // ~5MB/minute baseline
	}
	if baselineMbps <= 0 {
		baselineMbps = 10 // shape towards ~10Mbps typical home uplink
	}
	if geo == nil {
		geo = make(map[string]float64)
	}
	return &TrafficShaper{
		baseBytesPerMinute: baseBytesPerMinute,
		diurnal:            diurnal,
		geoFactors:         geo,
		baselineMbps:       baselineMbps,
		peerBuckets:        make(map[peer.ID]*bucket),
	}
}

// UpdateBaseline adjusts the shaping target to better match observed baseline network statistics.
func (ts *TrafficShaper) UpdateBaseline(mbps float64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if mbps > 0 {
		ts.baselineMbps = mbps
	}
}

// SetGeoFactor sets or updates a region-specific multiplier.
func (ts *TrafficShaper) SetGeoFactor(region string, factor float64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.geoFactors == nil {
		ts.geoFactors = make(map[string]float64)
	}
	ts.geoFactors[region] = factor
}

// Allow consumes tokens for the target peer if within the shaped budget.
// Region may be empty to skip geographic normalization.
func (ts *TrafficShaper) Allow(pid peer.ID, region string, payloadBytes int, now time.Time) bool {
	if payloadBytes <= 0 {
		return true
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	b := ts.bucketFor(pid)
	capBytes, refillPerSec := ts.derivedBudget(region, now)
	ts.refillBucket(b, refillPerSec, capBytes, now)

	if b.tokens < float64(payloadBytes) {
		return false
	}
	b.tokens -= float64(payloadBytes)
	return true
}

func (ts *TrafficShaper) bucketFor(pid peer.ID) *bucket {
	b, ok := ts.peerBuckets[pid]
	if !ok {
		b = &bucket{}
		ts.peerBuckets[pid] = b
	}
	return b
}

// derivedBudget returns capacity and refill rate after diurnal+geo shaping.
func (ts *TrafficShaper) derivedBudget(region string, now time.Time) (capacity float64, refillPerSec float64) {
	factor := ts.diurnal.Factor(now)
	if g, ok := ts.geoFactors[region]; ok && g > 0 {
		factor *= g
	}

	basePerMin := float64(ts.baseBytesPerMinute)
	capacity = basePerMin * factor
	refillPerSec = (basePerMin * factor) / 60.0

	// Fit to baseline Mbps ceiling (convert Mbps to bytes/sec).
	baselineBps := ts.baselineMbps * 125000
	refillPerSec = math.Min(refillPerSec, baselineBps)
	capacity = math.Min(capacity, baselineBps*60)
	return
}

func (ts *TrafficShaper) refillBucket(b *bucket, refillPerSec, capBytes float64, now time.Time) {
	if b.lastFill.IsZero() {
		b.tokens = capBytes
		b.lastFill = now
		return
	}
	if now.Before(b.lastFill) {
		b.lastFill = now
		return
	}
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens += refillPerSec * elapsed
	if b.tokens > capBytes {
		b.tokens = capBytes
	}
	b.lastFill = now
}

// ShapingDecider wraps an existing ForwardDecider with traffic shaping to enforce per-peer caps.
type ShapingDecider struct {
	inner   ForwardDecider
	shaper  *TrafficShaper
	regionf func(peer.ID) string
}

// NewShapingDecider composes shaping with an optional inner decider (can be nil).
func NewShapingDecider(inner ForwardDecider, shaper *TrafficShaper, regionLookup func(peer.ID) string) *ShapingDecider {
	return &ShapingDecider{inner: inner, shaper: shaper, regionf: regionLookup}
}

// AllowForward enforces both the inner policy and the shaping budget.
func (sd *ShapingDecider) AllowForward(target peer.ID, payloadBytes int) bool {
	if sd == nil || sd.shaper == nil {
		if sd != nil && sd.inner != nil {
			return sd.inner.AllowForward(target, payloadBytes)
		}
		return true
	}
	region := ""
	if sd.regionf != nil {
		region = sd.regionf(target)
	}
	if sd.inner != nil && !sd.inner.AllowForward(target, payloadBytes) {
		return false
	}
	return sd.shaper.Allow(target, region, payloadBytes, time.Now())
}
