// Project 1: API Client
//
// Spins up a small local HTTP server (via httptest, so this runs entirely
// offline and deterministically — no real network needed) serving the same
// data as both JSON and XML, then exercises a client against it: decoding
// responses, encoding a request body, and a custom UnmarshalJSON that
// accepts several different timestamp formats from "upstream" — a very
// common real-world API integration problem.
package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// FlexibleTime embeds time.Time, so it PROMOTES time.Time's own MarshalJSON
// (always RFC3339 out) but OVERRIDES UnmarshalJSON to accept several
// formats in. This is a deliberate asymmetry — see the README for why
// that's fine here, unlike Money in the CSV Importer project.
type FlexibleTime struct {
	time.Time
}

var acceptedLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	var lastErr error
	for _, layout := range acceptedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			ft.Time = t
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("unrecognized time format %q: %w", s, lastErr)
}

// User is the shape both JSON and XML responses decode into. Both sets of
// struct tags live on the SAME struct — nothing stops one type from
// supporting multiple serialization formats at once.
type User struct {
	XMLName   xml.Name     `json:"-" xml:"user"`
	ID        int          `json:"id" xml:"id,attr"`
	Name      string       `json:"name" xml:"name"`
	Email     string       `json:"email" xml:"email"`
	CreatedAt FlexibleTime `json:"createdAt" xml:"-"` // XML decoding of embedded
	                                                    // custom types needs more
	                                                    // setup than this demo
	                                                    // covers, so it's excluded
	                                                    // from the XML side on purpose.
}

type UserList struct {
	XMLName xml.Name `xml:"users"`
	Users   []User   `xml:"user"`
}

// --- Local server (stands in for a real API) -----------------------------

// seedUsers deliberately uses THREE different timestamp formats, simulating
// data that originated from different upstream systems — exactly the
// situation FlexibleTime's UnmarshalJSON exists to handle.
const usersJSON = `[
	{"id": 1, "name": "Ada Lovelace", "email": "ada@example.com", "createdAt": "2026-01-15T09:30:00Z"},
	{"id": 2, "name": "Grace Hopper", "email": "grace@example.com", "createdAt": "2026-02-20T14:00:00"},
	{"id": 3, "name": "Kemi Adeyemi", "email": "kemi@example.com", "createdAt": "2026-03-05"}
]`

func startServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(usersJSON))
	})

	mux.HandleFunc("GET /users.xml", func(w http.ResponseWriter, r *http.Request) {
		var users []User
		json.Unmarshal([]byte(usersJSON), &users) // reuse the same seed data
		list := UserList{Users: users}
		w.Header().Set("Content-Type", "application/xml")
		enc := xml.NewEncoder(w)
		enc.Indent("", "  ")
		enc.Encode(list)
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var newUser User
		if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		newUser.ID = 99
		newUser.CreatedAt = FlexibleTime{Time: time.Now()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newUser)
	})

	return httptest.NewServer(mux)
}

// --- Client -----------------------------------------------------

func fetchUsersJSON(baseURL string) ([]User, error) {
	resp, err := http.Get(baseURL + "/users")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var users []User
	// Streaming decode straight from the response body (Module 10's
	// streaming habit), rather than reading it all into a []byte first.
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("decoding JSON response: %w", err)
	}
	return users, nil
}

func fetchUsersXML(baseURL string) (*UserList, error) {
	resp, err := http.Get(baseURL + "/users.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list UserList
	if err := xml.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decoding XML response: %w", err)
	}
	return &list, nil
}

func createUser(baseURL string, name, email string) (*User, error) {
	reqBody, err := json.Marshal(map[string]string{"name": name, "email": email})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var created User
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("decoding created-user response: %w", err)
	}
	return &created, nil
}

func main() {
	server := startServer()
	defer server.Close()

	fmt.Println("=== Fetching users as JSON (three different timestamp formats) ===")
	users, err := fetchUsersJSON(server.URL)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, u := range users {
		fmt.Printf("  #%d %-15s %-25s created: %s\n", u.ID, u.Name, u.Email, u.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("\n=== Fetching the SAME data as XML ===")
	list, err := fetchUsersXML(server.URL)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, u := range list.Users {
		fmt.Printf("  #%d %-15s %s\n", u.ID, u.Name, u.Email)
	}

	fmt.Println("\n=== Creating a user (encoding a request body) ===")
	created, err := createUser(server.URL, "Tolu Bakare", "tolu@example.com")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("  created #%d %s at %s\n", created.ID, created.Name, created.CreatedAt.Format(time.RFC3339))
}
