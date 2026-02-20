package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T) (*httptest.Server, *store) {
	t.Helper()
	s := newStore()
	mux := newMux(s)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, s
}

func TestListItems(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/items")
	if err != nil {
		t.Fatalf("GET /api/items: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var items []Item
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != 1 || items[0].Name != "example" || items[0].Done != false {
		t.Errorf("unexpected seed item: %+v", items[0])
	}
}

func TestCreateItem(t *testing.T) {
	srv, _ := newTestServer(t)

	body := bytes.NewBufferString(`{"name":"new item","done":false}`)
	resp, err := http.Post(srv.URL+"/api/items", "application/json", body)
	if err != nil {
		t.Fatalf("POST /api/items: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if item.Name != "new item" {
		t.Errorf("expected name %q, got %q", "new item", item.Name)
	}
	if item.ID <= 0 {
		t.Errorf("expected positive ID, got %d", item.ID)
	}
}

func TestCreateItem_MissingName(t *testing.T) {
	srv, _ := newTestServer(t)

	body := bytes.NewBufferString(`{"done":false}`)
	resp, err := http.Post(srv.URL+"/api/items", "application/json", body)
	if err != nil {
		t.Fatalf("POST /api/items: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestCreateItem_InvalidJSON(t *testing.T) {
	srv, _ := newTestServer(t)

	body := bytes.NewBufferString(`not json`)
	resp, err := http.Post(srv.URL+"/api/items", "application/json", body)
	if err != nil {
		t.Fatalf("POST /api/items: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetItem(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/items/1")
	if err != nil {
		t.Fatalf("GET /api/items/1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if item.ID != 1 {
		t.Errorf("expected ID 1, got %d", item.ID)
	}
	if item.Name != "example" {
		t.Errorf("expected name %q, got %q", "example", item.Name)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/items/999")
	if err != nil {
		t.Fatalf("GET /api/items/999: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetItem_InvalidID(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/items/abc")
	if err != nil {
		t.Fatalf("GET /api/items/abc: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDeleteItem(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/items/1", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/items/1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify the item is gone.
	getResp, err := http.Get(srv.URL + "/api/items/1")
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getResp.StatusCode)
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/items/999", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/items/999: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestFullCRUDFlow(t *testing.T) {
	srv, _ := newTestServer(t)

	// 1. List — should have the seed item.
	resp, _ := http.Get(srv.URL + "/api/items")
	var items []Item
	json.NewDecoder(resp.Body).Decode(&items)
	resp.Body.Close()
	if len(items) != 1 {
		t.Fatalf("expected 1 seed item, got %d", len(items))
	}

	// 2. Create a new item.
	body := bytes.NewBufferString(`{"name":"task one","done":false}`)
	resp, _ = http.Post(srv.URL+"/api/items", "application/json", body)
	var created Item
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}

	// 3. List — should now have 2 items.
	resp, _ = http.Get(srv.URL + "/api/items")
	json.NewDecoder(resp.Body).Decode(&items)
	resp.Body.Close()
	if len(items) != 2 {
		t.Fatalf("expected 2 items after create, got %d", len(items))
	}

	// 4. Get the created item by ID.
	resp, _ = http.Get(srv.URL + "/api/items/" + itoa(created.ID))
	var fetched Item
	json.NewDecoder(resp.Body).Decode(&fetched)
	resp.Body.Close()
	if fetched.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, fetched.ID)
	}

	// 5. Delete the created item.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/items/"+itoa(created.ID), nil)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("delete: expected 200, got %d", resp.StatusCode)
	}

	// 6. List — back to 1 item.
	resp, _ = http.Get(srv.URL + "/api/items")
	json.NewDecoder(resp.Body).Decode(&items)
	resp.Body.Close()
	if len(items) != 1 {
		t.Fatalf("expected 1 item after delete, got %d", len(items))
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
