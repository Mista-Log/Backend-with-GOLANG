// Todo API — built with Gin, covering CRUD, request validation via
// `binding` struct tags, offset-based pagination, and Swagger/OpenAPI
// annotations (see each handler's godoc comment in handlers.go).
//
// Run with: go mod tidy && go run .
// (go mod tidy needs internet access to fetch Gin — see README.md)
package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	store := NewTodoStore()
	store.Create("Learn Gin")
	store.Create("Add Swagger docs")
	store.Create("Ship the Todo API")

	// gin.Default() includes Logger and Recovery middleware BUILT IN —
	// the same logging/panic-recovery concepts from Module 15's guide,
	// provided out of the box instead of hand-written.
	r := gin.Default()

	r.GET("/todos", listTodos(store))
	r.GET("/todos/:id", getTodo(store))
	r.POST("/todos", createTodo(store))
	r.PUT("/todos/:id", replaceTodo(store))
	r.PATCH("/todos/:id", patchTodo(store))
	r.DELETE("/todos/:id", deleteTodo(store))

	fmt.Println("Todo API (Gin) listening on http://localhost:8080")
	fmt.Println("See README.md for a full curl walkthrough.")
	r.Run(":8080")
}
