package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type EventHandler struct{ service *service.EventService }

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
	payloadValues := form.Value["payload"]
	if len(payloadValues) == 0 {
		return req, fmt.Errorf("payload is required")
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
	for field, headers := range form.File {
		if len(headers) == 0 {
			continue
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
		blobs[field] = fileBlob{name: fh.Filename, content: content}
	}

	// Prefer explicit file_evidence metadata from payload.
	if len(req.FileEvidence) > 0 {
		for i := range req.FileEvidence {
			field := req.FileEvidence[i].Field
			if field == "" {
				field = "file_" + strconv.Itoa(req.FileEvidence[i].EventIndex) + "_" + strconv.Itoa(i)
			}
			blob, ok := blobs[field]
			if !ok {
				return req, fmt.Errorf("missing multipart file: %s", field)
			}
			req.FileEvidence[i].Content = blob.content
			if req.FileEvidence[i].Filename == "" {
				req.FileEvidence[i].Filename = blob.name
			}
		}
		return req, nil
	}

	// Auto-map file_{event}_{n} fields when payload omits file_evidence.
	for field, blob := range blobs {
		if !strings.HasPrefix(field, "file_") {
			continue
		}
		parts := strings.Split(field, "_")
		if len(parts) < 3 {
			continue
		}
		eventIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
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
	if e.Status != "published" {
		userID, _ := c.Get("user_id").(string)
		roles, _ := c.Get("roles").([]string)
		privileged := false
		for _, role := range roles {
			if role == "admin" || role == "moderator" {
				privileged = true
				break
			}
		}
		if !privileged && (e.SubmittedBy == nil || *e.SubmittedBy != userID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
		}
	}
	return c.JSON(http.StatusOK, e)
}
