package api

import (
	"net/http"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type ListHandler struct {
	listService ListService
}

type ListService interface {
	GetListsByUserID(userID string) ([]TaskListWithMembers, error)
	CreateList(list models.TaskList) (*models.TaskList, error)
	GetListByID(listID string) (*models.TaskList, error)
	UpdateList(list models.TaskList) (*models.TaskList, error)
	DeleteList(listID string, userID string) error
	GetListMembers(listID string) ([]models.ListMember, error)
	AddListMember(member models.ListMember) (*models.ListMember, error)
}

type TaskListWithMembers struct {
	models.TaskList
	Members []models.ListMember `json:"members"`
}

type ListCreateRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Color       string  `json:"color"`
	Icon        string  `json:"icon"`
	IsShared    bool    `json:"is_shared"`
	ParentID    *string `json:"parent_id"`
}

func NewListHandler(listService ListService) *ListHandler {
	return &ListHandler{
		listService: listService,
	}
}

// GetLists handles GET /lists - get user's task lists with sharing info
func (h *ListHandler) GetLists(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	lists, err := h.listService.GetListsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get task lists",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"lists": lists,
		"total": len(lists),
	})
}

// CreateList handles POST /lists - create new task list
func (h *ListHandler) CreateList(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	var req ListCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Create task list model using NewTaskList constructor for validation
	taskList, err := models.NewTaskList(req.Name, req.Description, user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid task list data",
			Details: err.Error(),
		})
		return
	}

	// Set optional fields
	if req.Color != "" {
		if err := taskList.SetColor(req.Color); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid color format",
				Details: err.Error(),
			})
			return
		}
	}

	if req.Icon != "" {
		taskList.SetIcon(req.Icon)
	}

	if req.IsShared {
		taskList.IsShared = req.IsShared
	}

	if req.ParentID != nil {
		if err := taskList.SetParent(*req.ParentID); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid parent list",
				Details: err.Error(),
			})
			return
		}
	}

	// Create task list
	createdList, err := h.listService.CreateList(*taskList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create task list",
		})
		return
	}

	c.JSON(http.StatusCreated, createdList)
}