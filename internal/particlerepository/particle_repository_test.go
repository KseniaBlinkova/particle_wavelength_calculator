package particlerepository

import "testing"

func TestGetPublishedParticleByIDDoesNotExposeOtherStatuses(t *testing.T) {
	if particle := GetPublishedParticleByID(1); particle == nil {
		t.Fatal("published particle was not found")
	}
	if particle := GetPublishedParticleByID(3); particle != nil {
		t.Fatalf("draft particle was returned: %+v", particle)
	}
	if particle := GetPublishedParticleByID(4); particle != nil {
		t.Fatalf("deleted particle was returned: %+v", particle)
	}
}

func TestGetParticleByIDReturnsSafeCopy(t *testing.T) {
	particle := GetParticleByID(1)
	if particle == nil {
		t.Fatal("particle was not found")
	}

	originalName := particle.Name
	particle.Name = "changed outside repository"
	stored := GetParticleByID(1)
	if stored == nil || stored.Name != originalName {
		t.Fatal("changing returned particle changed repository data")
	}
}
