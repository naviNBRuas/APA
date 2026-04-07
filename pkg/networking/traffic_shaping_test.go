package networking

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestTrafficShaperCapsPerPeer(t *testing.T) {
	diurnal := DiurnalCurve{}
	ts := NewTrafficShaper(60000, diurnal, nil, 50)
	now := time.Now()

	pid := peer.ID("peer-a")
	if !ts.Allow(pid, "", 30000, now) {
		t.Fatalf("expected first allowance within cap")
	}
	if !ts.Allow(pid, "", 30000, now) {
		t.Fatalf("expected second allowance within cap")
	}
	if ts.Allow(pid, "", 1000, now) {
		t.Fatalf("expected cap exceeded for peer")
	}

	// Another peer has its own bucket.
	if !ts.Allow(peer.ID("peer-b"), "", 30000, now) {
		t.Fatalf("expected separate bucket per peer")
	}
}

func TestTrafficShaperDiurnalAndGeo(t *testing.T) {
	diurnal := DiurnalCurve{}
	diurnal.Factors[10] = 0.5 // halve traffic at 10:00
	geo := map[string]float64{"eu": 0.5}
	ts := NewTrafficShaper(60000, diurnal, geo, 50)

	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	pid := peer.ID("peer-diurnal")

	// With diurnal 0.5, effective cap is 30k.
	if !ts.Allow(pid, "", 20000, base) {
		t.Fatalf("expected allowance under diurnal cap")
	}
	if ts.Allow(pid, "", 15000, base) {
		t.Fatalf("expected cap hit after diurnal budget used")
	}

	// Geographic normalization further halves budget to ~30k -> 15k available for new peer.
	pid2 := peer.ID("peer-geo")
	if !ts.Allow(pid2, "eu", 15000, base) {
		t.Fatalf("expected allowance under geo-normalized cap")
	}
	if ts.Allow(pid2, "eu", 15000, base) {
		t.Fatalf("expected geo-normalized cap exceeded")
	}
}

type denyOnceDecider struct{ called int }

func (d *denyOnceDecider) AllowForward(peer.ID, int) bool {
	d.called++
	return d.called > 1
}

func TestShapingDeciderComposes(t *testing.T) {
	diurnal := DiurnalCurve{}
	ts := NewTrafficShaper(30000, diurnal, nil, 50)
	inner := &denyOnceDecider{}
	sd := NewShapingDecider(inner, ts, nil)

	pid := peer.ID("peer-test")
	if sd.AllowForward(pid, 10000) {
		t.Fatalf("expected inner decider to deny first")
	}
	if !sd.AllowForward(pid, 10000) {
		t.Fatalf("expected allowance after inner permits")
	}
	// Exhaust remaining 20k of 30k cap, next should fail.
	if !sd.AllowForward(pid, 15000) {
		t.Fatalf("expected second allowance until cap is near limit")
	}
	if sd.AllowForward(pid, 15000) {
		t.Fatalf("expected shaping bucket to cap out")
	}
}
