package simulator

import (
	"math/rand"
	"testing"
)

func TestChargerCodeNormalizesPrefix(t *testing.T) {
	got := chargerCode(" sim ac ", "20260504", 7)
	want := "SIM-AC-20260504-007"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestConnectorCode(t *testing.T) {
	got := connectorCode("SIM-001", 2)
	want := "SIM-001-02"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestPowerForBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, curve := range []string{LoadFlat, LoadCommute, LoadRandom} {
		for tick := 0; tick < 10; tick++ {
			power := powerFor(curve, tick, 10, 7, rng)
			if power < 0 || power > 7 {
				t.Fatalf("curve %s produced out-of-range power %.2f", curve, power)
			}
		}
	}
}
