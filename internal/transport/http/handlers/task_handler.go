package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TaskHandler struct {
	usecase taskusecase.Usecase
}

func NewTaskHandler(usecase taskusecase.Usecase) *TaskHandler {
	return &TaskHandler{usecase: usecase}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), taskusecase.CreateInput{
		Title:                  req.Title,
		Description:            req.Description,
		Status:                 req.Status,
		Recurrence:             req.Recurrence,
		MaterializeHorizonDays: req.MaterializeHorizonDays,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTaskDTO(created))
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	task, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(task))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req updateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, taskusecase.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Recurrence:  req.Recurrence,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := listFilterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tasks, err := h.usecase.List(r.Context(), filter)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	response := make([]taskDTO, 0, len(tasks))
	for i := range tasks {
		response = append(response, newTaskDTO(&tasks[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *TaskHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req materializeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	from, err := time.ParseInLocation("2006-01-02", req.From, time.UTC)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("from must be YYYY-MM-DD"))
		return
	}

	to, err := time.ParseInLocation("2006-01-02", req.To, time.UTC)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("to must be YYYY-MM-DD"))
		return
	}

	created, err := h.usecase.Materialize(r.Context(), id, from, to)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, materializeResponse{Created: created})
}

func listFilterFromQuery(r *http.Request) (taskdomain.ListTasksFilter, error) {
	q := r.URL.Query()

	filter := taskdomain.ListTasksFilter{}

	if raw := q.Get("include_templates"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return taskdomain.ListTasksFilter{}, errors.New("include_templates must be a boolean")
		}

		filter.IncludeTemplates = v
	}

	if raw := q.Get("occurrence_from"); raw != "" {
		t, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
		if err != nil {
			return taskdomain.ListTasksFilter{}, errors.New("occurrence_from must be YYYY-MM-DD")
		}

		filter.OccurrenceFrom = &t
	}

	if raw := q.Get("occurrence_to"); raw != "" {
		t, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
		if err != nil {
			return taskdomain.ListTasksFilter{}, errors.New("occurrence_to must be YYYY-MM-DD")
		}

		filter.OccurrenceTo = &t
	}

	return filter, nil
}

func getIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing task id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid task id")
	}

	if id <= 0 {
		return 0, errors.New("invalid task id")
	}

	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskusecase.ErrNotATemplate):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, taskdomain.ErrInvalidRecurrence):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, taskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
