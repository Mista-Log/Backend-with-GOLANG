package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateTodoRequest and UpdateTodoRequest use Gin's `binding` tags —
// go-playground/validator runs automatically inside ShouldBindJSON,
// exactly as described in the guide's Validation section.
type CreateTodoRequest struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
}

type UpdateTodoRequest struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
	Done  bool   `json:"done"`
}

type PatchTodoRequest struct {
	Done *bool `json:"done" binding:"required"` // pointer so `false` is distinguishable from "not sent"
}

type PaginatedResponse struct {
	Data       []*Todo `json:"data"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	Total      int     `json:"total"`
	TotalPages int     `json:"totalPages"`
}

// listTodos godoc
// @Summary      List todos
// @Description  Returns a paginated list of todos
// @Tags         todos
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Items per page (default 10)"
// @Success      200    {object}  PaginatedResponse
// @Router       /todos [get]
func listTodos(store *TodoStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		todos, total := store.List(page, limit)
		totalPages := (total + limit - 1) / limit // ceiling division

		c.JSON(http.StatusOK, PaginatedResponse{
			Data:       todos,
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		})
	}
}

// getTodo godoc
// @Summary      Get a todo by ID
// @Tags         todos
// @Produce      json
// @Param        id   path      int  true  "Todo ID"
// @Success      200  {object}  Todo
// @Failure      404  {object}  map[string]string
// @Router       /todos/{id} [get]
func getTodo(store *TodoStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		todo, ok := store.Get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
			return
		}
		c.JSON(http.StatusOK, todo)
	}
}

// createTodo godoc
// @Summary      Create a todo
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        todo  body      CreateTodoRequest  true  "Todo to create"
// @Success      201   {object}  Todo
// @Failure      400   {object}  map[string]string
// @Router       /todos [post]
func createTodo(store *TodoStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTodoRequest
		// ShouldBindJSON runs validation from the `binding` tags
		// automatically — a request with an empty or 300-character Title
		// never reaches this handler's logic below at all.
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		todo := store.Create(req.Title)
		c.JSON(http.StatusCreated, todo)
	}
}

// replaceTodo godoc
// @Summary      Replace a todo (PUT semantics)
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        id    path      int                true  "Todo ID"
// @Param        todo  body      UpdateTodoRequest  true  "Full replacement"
// @Success      200   {object}  Todo
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /todos/{id} [put]
func replaceTodo(store *TodoStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req UpdateTodoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		todo, err := store.Replace(id, req.Title, req.Done)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, todo)
	}
}

// patchTodo godoc
// @Summary      Mark a todo done/undone (PATCH semantics)
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        id     path      int               true  "Todo ID"
// @Param        patch  body      PatchTodoRequest  true  "Partial update"
// @Success      200    {object}  Todo
// @Failure      400    {object}  map[string]string
// @Failure      404    {object}  map[string]string
// @Router       /todos/{id} [patch]
func patchTodo(store *TodoStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req PatchTodoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// req.Done is guaranteed non-nil here — `binding:"required"` on a
		// *bool rejects a request that omits "done" entirely, while still
		// allowing an EXPLICIT `"done": false` to come through correctly.
		todo, err := store.PatchDone(id, *req.Done)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, todo)
	}
}

// deleteTodo godoc
// @Summary      Delete a todo
// @Tags         todos
// @Param        id   path  int  true  "Todo ID"
// @Success      204
// @Failure      404  {object}  map[string]string
// @Router       /todos/{id} [delete]
func deleteTodo(store *TodoStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := store.Delete(id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
