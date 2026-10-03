package commerce

import "testing"

func TestDecide(t *testing.T) {
	tests := []struct {
		name      string
		update    Update
		state     string
		rate      float64
		netChange int
	}{
		{"ready asset", Update{EventID: "evt-1", CreatorID: "creator-7", AssetID: "asset-9", DeliveryAttempts: 10, SuccessfulDeliveries: 9, SubscriberAdds: 12, SubscriberCancels: 2, ProcessingLagMS: 840, ProcessingComplete: true, OccurredAt: "2026-09-28T10:00:00Z"}, "ready", 0.9, 10},
		{"processing asset", Update{EventID: "evt-2", CreatorID: "creator-7", AssetID: "asset-10", SubscriberAdds: 1, SubscriberCancels: 3, ProcessingLagMS: 120, OccurredAt: "2026-09-28T10:01:00Z"}, "processing", 0, -2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Decide(test.update)
			if err != nil {
				t.Fatal(err)
			}
			if got.AssetState != test.state || got.DeliverySuccessRate != test.rate || got.SubscriberNetChange != test.netChange {
				t.Fatalf("got state=%s rate=%v net=%d", got.AssetState, got.DeliverySuccessRate, got.SubscriberNetChange)
			}
			if len(got.Points) != 3 {
				t.Fatalf("got %d metric points", len(got.Points))
			}
		})
	}
}
