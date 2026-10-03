package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/infrai-examples/creator-operations-stream/internal/commerce"
	"github.com/infrai-examples/creator-operations-stream/internal/infrai"
)

const channel = "creator-operations"

type server struct{ client *infrai.Client }

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := infrai.New(key)
	s := &server{client: client}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /updates", s.update)
	mux.HandleFunc("POST /dashboard-token", s.dashboardToken)
	log.Println("creator dashboard ingestion listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) update(w http.ResponseWriter, r *http.Request) {
	var input commerce.Update
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	snapshot, err := commerce.Decide(input)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if err := s.client.SendSnapshot(r.Context(), channel, snapshot); err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, snapshot)
}

func (s *server) dashboardToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); err != nil || input.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}
	data, err := s.client.IssueDashboardToken(r.Context(), channel, input.ClientID)
	if err != nil {
		writeClientError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeClientError(w http.ResponseWriter, err error) {
	var apiErr *infrai.APIError
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		writeJSON(w, apiErr.Status, map[string]string{"error": apiErr.Code, "message": apiErr.Message})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
