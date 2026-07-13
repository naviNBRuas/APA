package networking

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestTrafficShaperCapsPerPeer(t *testing.T) {
	diurnal := DiurnalCurve{}
	ts := NewTrafficShaper(60000, diurnal, nil, 50)
	now := time.Now()

	pid := peer.ID("peer-a")
	require.True(t, ts.Allow(pid, "", 30000, now), "expected first allowance within cap")
	require.True(t, ts.Allow(pid, "", 30000, now), "expected second allowance within cap")
	require.False(t, ts.Allow(pid, "", 1000, now), "expected cap exceeded for peer")

	require.True(t, ts.Allow(peer.ID("peer-b"), "", 30000, now), "expected separate bucket per peer")
}

func TestTrafficShaperDiurnalAndGeo(t *testing.T) {
	diurnal := DiurnalCurve{}
	diurnal.Factors[10] = 0.5 // halve traffic at 10:00
	geo := map[string]float64{"eu": 0.5}
	ts := NewTrafficShaper(60000, diurnal, geo, 50)

	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	pid := peer.ID("peer-diurnal")

	require.True(t, ts.Allow(pid, "", 20000, base), "expected allowance under diurnal cap")
	require.False(t, ts.Allow(pid, "", 15000, base), "expected cap hit after diurnal budget used")

	pid2 := peer.ID("peer-geo")
	require.True(t, ts.Allow(pid2, "eu", 15000, base), "expected allowance under geo-normalized cap")
	require.False(t, ts.Allow(pid2, "eu", 15000, base), "expected geo-normalized cap exceeded")
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
	require.False(t, sd.AllowForward(pid, 10000), "expected inner decider to deny first")
	require.True(t, sd.AllowForward(pid, 10000), "expected allowance after inner permits")
	require.True(t, sd.AllowForward(pid, 15000), "expected second allowance until cap is near limit")
	require.False(t, sd.AllowForward(pid, 15000), "expected shaping bucket to cap out")
}
