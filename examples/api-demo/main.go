package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// Item represents a single item in the collection.
type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Done bool   `json:"done"`
}

// store is a simple in-memory store with mutex protection.
type store struct {
	mu     sync.Mutex
	items  map[int]Item
	nextID int
}

func newStore() *store {
	s := &store{
		items:  make(map[int]Item),
		nextID: 2,
	}
	s.items[1] = Item{ID: 1, Name: "example", Done: false}
	return s
}

func (s *store) list() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, item)
	}
	// Sort by ID for deterministic output.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].ID > result[j].ID {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func (s *store) get(id int) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	return item, ok
}

func (s *store) create(name string, done bool) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := Item{ID: s.nextID, Name: name, Done: done}
	s.items[s.nextID] = item
	s.nextID++
	return item
}

func (s *store) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}

// newMux builds the HTTP mux for the API and returns it.
// This is a separate function so tests can call it directly without binding a port.
func newMux(s *store) *http.ServeMux {
	mux := http.NewServeMux()

	// GET /api/items
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) {
		items := s.list()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(items); err != nil {
			log.Printf("encode list: %v", err)
		}
	})

	// POST /api/items
	mux.HandleFunc("POST /api/items", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
			Done bool   `json:"done"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if body.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusUnprocessableEntity)
			return
		}
		item := s.create(body.Name, body.Done)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(item); err != nil {
			log.Printf("encode create: %v", err)
		}
	})

	// GET /api/items/{id}
	mux.HandleFunc("GET /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}
		item, ok := s.get(id)
		if !ok {
			http.Error(w, fmt.Sprintf(`{"error":"item %d not found"}`, id), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(item); err != nil {
			log.Printf("encode get: %v", err)
		}
	})

	// DELETE /api/items/{id}
	mux.HandleFunc("DELETE /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}
		if !s.delete(id) {
			http.Error(w, fmt.Sprintf(`{"error":"item %d not found"}`, id), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"deleted":%d}`, id)
	})

	return mux
}

func main() {
	s := newStore()
	mux := newMux(s)
	addr := ":8080"
	log.Printf("api-demo listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
