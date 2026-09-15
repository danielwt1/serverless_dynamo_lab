package handlers

import (
	"encoding/json"
	"errors"
	domainerrors "get_task_lambda/internal/domain/errors"
	"get_task_lambda/internal/domain/in"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type TaskHandlers struct {
	usecase in.TasksQueryApi
}

func NewTaskHandlers(usecase in.TasksQueryApi) *TaskHandlers {
	return &TaskHandlers{usecase}
}

func (h *TaskHandlers) GetTasksByUser(w http.ResponseWriter, r *http.Request) {
	userId := strings.TrimSpace(chi.URLParam(r, "ownerId"))
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if userId == "" {
		h.writeResponse(w, map[string]string{"message": "ownerId es obligatorio"}, http.StatusBadRequest)
		return
	}
	response, err := h.usecase.FindTasks(r.Context(), userId, cursor)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeResponse(w, response, http.StatusOK)

}

func (h *TaskHandlers) GetPendingTasksByUserId(w http.ResponseWriter, r *http.Request) {
	userId := strings.TrimSpace(chi.URLParam(r, "ownerId"))
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if userId == "" {
		h.writeResponse(w, map[string]string{"message": "ownerId es obligatorio"}, http.StatusBadRequest)
		return
	}
	response, err := h.usecase.FindPendingTasksByUser(r.Context(), userId, cursor)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeResponse(w, response, http.StatusOK)
}

func (h *TaskHandlers) writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, domainerrors.InvalidCursor) {
		h.writeResponse(w, map[string]string{"message": "cursor inválido"}, http.StatusBadRequest)
		return
	}
	h.writeResponse(w, map[string]string{"message": "no se pudieron consultar las tareas"}, http.StatusInternalServerError)
}

func (h *TaskHandlers) writeResponse(w http.ResponseWriter, response interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// At this point the HTTP status was already sent. The best action is to
		// let the Lambda adapter return the response that was written so far.
		return
	}
}
