// CLI Todo — a persistent, crash-safe command-line task manager.
//
// Usage:
//
//	todo add "Buy milk"
//	todo list
//	todo done 2
//	todo rm 2
//
// Run with: go run . add "Buy milk"
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

const dataFile = "todo.json"

func loadTasks() ([]Task, error) {
	data, err := os.ReadFile(dataFile)
	if os.IsNotExist(err) {
		return nil, nil // no file yet — an empty todo list, not an error
	}
	if err != nil {
		return nil, err
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("todo.json is corrupted: %w", err)
	}
	return tasks, nil
}

// saveTasks uses Module 10's safe-write pattern — write to a temp file in
// the SAME directory (so the later rename is guaranteed atomic, same
// filesystem), then rename over the real file. If the program is killed
// mid-write, todo.json is untouched; the temp file is simply an orphan,
// never a half-written replacement for the real data.
func saveTasks(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(dataFile)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, "todo-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op if the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, dataFile)
}

func nextID(tasks []Task) int {
	max := 0
	for _, t := range tasks {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1
}

func cmdAdd(title string) error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	tasks = append(tasks, Task{ID: nextID(tasks), Title: title, CreatedAt: time.Now()})
	if err := saveTasks(tasks); err != nil {
		return err
	}
	fmt.Println("added:", title)
	return nil
}

func cmdList() error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Println(`no tasks yet — try: todo add "Buy milk"`)
		return nil
	}
	for _, t := range tasks {
		mark := " "
		if t.Done {
			mark = "x"
		}
		fmt.Printf("[%s] %d. %s\n", mark, t.ID, t.Title)
	}
	return nil
}

func cmdDone(id int) error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	found := false
	for i := range tasks {
		if tasks[i].ID == id {
			tasks[i].Done = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no task with id %d", id)
	}
	if err := saveTasks(tasks); err != nil {
		return err
	}
	fmt.Println("marked done:", id)
	return nil
}

func cmdRemove(id int) error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	out := tasks[:0] // reuse the backing array — no new allocation needed
	found := false
	for _, t := range tasks {
		if t.ID == id {
			found = true
			continue
		}
		out = append(out, t)
	}
	if !found {
		return fmt.Errorf("no task with id %d", id)
	}
	if err := saveTasks(out); err != nil {
		return err
	}
	fmt.Println("removed:", id)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: todo <add|list|done|rm> [args]")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "add":
		if len(os.Args) < 3 {
			err = fmt.Errorf(`usage: todo add "<title>"`)
		} else {
			err = cmdAdd(os.Args[2])
		}
	case "list":
		err = cmdList()
	case "done":
		var id int
		if len(os.Args) < 3 {
			err = fmt.Errorf("usage: todo done <id>")
		} else {
			fmt.Sscanf(os.Args[2], "%d", &id)
			err = cmdDone(id)
		}
	case "rm":
		var id int
		if len(os.Args) < 3 {
			err = fmt.Errorf("usage: todo rm <id>")
		} else {
			fmt.Sscanf(os.Args[2], "%d", &id)
			err = cmdRemove(id)
		}
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
