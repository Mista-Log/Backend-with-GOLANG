package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type taskRequest struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	// Header().Set MUST happen before WriteHeader — Module 15's Response
	// section, section 3's "one-way door" diagram.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func listTasksHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.List())
	}
}

func getTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id")) // Go 1.22+ routing — Section 5
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}
		task, ok := store.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSON(w, http.StatusOK, task)
	}
}

func createTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req taskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		task := store.Create(req.Title)
		writeJSON(w, http.StatusCreated, task)
	}
}

func updateTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}
		var req taskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		task, err := store.Update(id, req.Title, req.Done)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, task)
	}
}

func deleteTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}
		if err := store.Delete(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent) // 204 — success, deliberately no body
	}
}
