package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/service"
)

type eventService interface {
	Publish(context.Context, service.PublishSubjectRequest, string) (*models.Subject, error)
	Get(context.Context, string) (*models.Event, error)
	GetVersions(context.Context, string) ([]models.EventVersion, error)
}

type EventHandler struct{ service eventService }

func NewEventHandler(s *service.EventService) *EventHandler { return &EventHandler{service: s} }

func (h *EventHandler) Publish(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	req, err := bindPublishRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	subject, err := h.service.Publish(c.Request().Context(), req, userID)
	if err != nil {
		if err == service.ErrSubmissionRestricted {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, subject)
}

func bindPublishRequest(c echo.Context) (service.PublishSubjectRequest, error) {
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 72<<20)
	ct := c.Request().Header.Get(echo.HeaderContentType)
	if strings.HasPrefix(ct, echo.MIMEMultipartForm) {
		return bindMultipartPublish(c)
	}
	var req service.PublishSubjectRequest
	if err := c.Bind(&req); err != nil {
		return req, err
	}
	return req, nil
}

// bindMultipartPublish accepts:
//   - payload: JSON PublishSubjectRequest (file_evidence entries may set field names)
//   - files named file_{eventIndex}_{n} or referenced by file_evidence[].field
func bindMultipartPublish(c echo.Context) (service.PublishSubjectRequest, error) {
	var req service.PublishSubjectRequest
	form, err := c.MultipartForm()
	if err != nil {
		return req, err
	}
	defer form.RemoveAll()
	payloadValues := form.Value["payload"]
	if len(payloadValues) != 1 {
		return req, fmt.Errorf("exactly one payload is required")
	}
	if err := json.Unmarshal([]byte(payloadValues[0]), &req); err != nil {
		return req, fmt.Errorf("invalid payload json")
	}

	// Collect multipart files by field name.
	type fileBlob struct {
		name    string
		content []byte
	}
	blobs := map[string]fileBlob{}
	var totalBytes int
	if len(form.File) > 64 {
		return req, fmt.Errorf("too many evidence files")
	}
	for field, headers := range form.File {
		if len(headers) == 0 {
			continue
		}
		if len(headers) != 1 {
			return req, fmt.Errorf("duplicate multipart file field: %s", field)
		}
		fh := headers[0]
		src, err := fh.Open()
		if err != nil {
			return req, err
		}
		content, err := io.ReadAll(io.LimitReader(src, int64(service.MaxPublishFileBytes)+1))
		src.Close()
		if err != nil {
			return req, err
		}
		if len(content) > service.MaxPublishFileBytes {
			return req, fmt.Errorf("file too large")
		}
		totalBytes += len(content)
		if totalBytes > 64<<20 {
			return req, fmt.Errorf("total evidence files too large")
		}
		blobs[field] = fileBlob{name: fh.Filename, content: content}
	}

	// Prefer explicit file_evidence metadata from payload.
	if len(req.FileEvidence) > 0 {
		used := make(map[string]bool)
		for i := range req.FileEvidence {
			field := req.FileEvidence[i].Field
			if field == "" {
				field = "file_" + strconv.Itoa(req.FileEvidence[i].EventIndex) + "_" + strconv.Itoa(i)
			}
			if used[field] {
				return req, fmt.Errorf("duplicate file reference: %s", field)
			}
			used[field] = true
			blob, ok := blobs[field]
			if !ok {
				return req, fmt.Errorf("missing multipart file: %s", field)
			}
			req.FileEvidence[i].Content = blob.content
			if req.FileEvidence[i].Filename == "" {
				req.FileEvidence[i].Filename = blob.name
			}
		}
		if len(used) != len(blobs) {
			return req, fmt.Errorf("unreferenced multipart file")
		}
		return req, nil
	}

	// Auto-map file_{event}_{n} fields when payload omits file_evidence.
	for field, blob := range blobs {
		parts := strings.Split(field, "_")
		if len(parts) != 3 || parts[0] != "file" {
			return req, fmt.Errorf("invalid multipart file field")
		}
		eventIndex, err := strconv.Atoi(parts[1])
		if err != nil || eventIndex < 0 {
			return req, fmt.Errorf("invalid multipart event index")
		}
		fileIndex, err := strconv.Atoi(parts[2])
		if err != nil || fileIndex < 0 {
			return req, fmt.Errorf("invalid multipart file index")
		}
		req.FileEvidence = append(req.FileEvidence, service.PublishFileEvidenceRequest{
			EventIndex: eventIndex,
			Title:      blob.name,
			Filename:   blob.name,
			Content:    blob.content,
			Field:      field,
		})
	}
	return req, nil
}

func (h *EventHandler) Get(c echo.Context) error {
	e, err := h.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	if !eventVisibleToRequester(c, e) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	return c.JSON(http.StatusOK, e)
}

func (h *EventHandler) GetVersions(c echo.Context) error {
	// Authorize the current event before loading historical snapshots, including
	// when the version repository is disabled or the history is empty.
	event, err := h.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil || !eventVisibleToRequester(c, event) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	versions, err := h.service.GetVersions(c.Request().Context(), c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	if versions == nil {
		versions = []models.EventVersion{}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"versions": versions})
}
