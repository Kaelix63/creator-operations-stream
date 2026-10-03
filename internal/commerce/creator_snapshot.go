package commerce

import "fmt"

type Update struct {
	EventID              string `json:"event_id"`
	CreatorID            string `json:"creator_id"`
	AssetID              string `json:"asset_id"`
	DeliveryAttempts     int    `json:"delivery_attempts"`
	SuccessfulDeliveries int    `json:"successful_deliveries"`
	SubscriberAdds       int    `json:"subscriber_adds"`
	SubscriberCancels    int    `json:"subscriber_cancels"`
	ProcessingLagMS      int    `json:"processing_lag_ms"`
	ProcessingComplete   bool   `json:"processing_complete"`
	OccurredAt           string `json:"occurred_at"`
}

type Point struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Type      string            `json:"type"`
	Tags      map[string]string `json:"tags"`
	Timestamp string            `json:"timestamp"`
}

type Snapshot struct {
	EventID             string  `json:"event_id"`
	CreatorID           string  `json:"creator_id"`
	AssetID             string  `json:"asset_id"`
	AssetState          string  `json:"asset_state"`
	DeliverySuccessRate float64 `json:"delivery_success_rate"`
	SubscriberNetChange int     `json:"subscriber_net_change"`
	ProcessingLagMS     int     `json:"processing_lag_ms"`
	OccurredAt          string  `json:"occurred_at"`
	Points              []Point `json:"-"`
}

func Decide(update Update) (Snapshot, error) {
	if update.EventID == "" || update.CreatorID == "" || update.AssetID == "" || update.OccurredAt == "" {
		return Snapshot{}, fmt.Errorf("event_id, creator_id, asset_id, and occurred_at are required")
	}
	if update.DeliveryAttempts < 0 || update.SuccessfulDeliveries < 0 || update.SuccessfulDeliveries > update.DeliveryAttempts {
		return Snapshot{}, fmt.Errorf("delivery counts are inconsistent")
	}
	if update.SubscriberAdds < 0 || update.SubscriberCancels < 0 || update.ProcessingLagMS < 0 {
		return Snapshot{}, fmt.Errorf("counts and processing lag must be non-negative")
	}

	rate := 0.0
	if update.DeliveryAttempts > 0 {
		rate = float64(update.SuccessfulDeliveries) / float64(update.DeliveryAttempts)
	}
	state := "processing"
	if update.ProcessingComplete {
		state = "ready"
	}
	net := update.SubscriberAdds - update.SubscriberCancels
	tags := map[string]string{"asset_state": state}
	snapshot := Snapshot{
		EventID: update.EventID, CreatorID: update.CreatorID, AssetID: update.AssetID,
		AssetState: state, DeliverySuccessRate: rate, SubscriberNetChange: net,
		ProcessingLagMS: update.ProcessingLagMS, OccurredAt: update.OccurredAt,
	}
	snapshot.Points = []Point{
		{Name: "creator.delivery.success_rate", Value: rate, Type: "gauge", Tags: tags, Timestamp: update.OccurredAt},
		{Name: "creator.subscriber.net_change", Value: float64(net), Type: "gauge", Tags: tags, Timestamp: update.OccurredAt},
		{Name: "creator.content.processing_lag_ms", Value: float64(update.ProcessingLagMS), Type: "timing", Tags: tags, Timestamp: update.OccurredAt},
	}
	return snapshot, nil
}
