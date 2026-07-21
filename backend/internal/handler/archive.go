package handler

import (
	"bytes"
	"net/http"

	"github.com/labstack/echo/v4"

	exporter "github.com/ZhX589/UniBlack/backend/internal/export"
)

type ArchiveHandler struct{ service *exporter.ArchiveService }

func NewArchiveHandler(s *exporter.ArchiveService) *ArchiveHandler {
	return &ArchiveHandler{service: s}
}
func (h *ArchiveHandler) Export(c echo.Context) error {
	b, err := h.service.Build(c.Request().Context(), c.Param("publicID"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.Blob(http.StatusOK, "application/zip", b)
}
func (h *ArchiveHandler) PreviewImport(c echo.Context) error {
	file, err := c.FormFile("archive")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "archive is required"})
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid archive"})
	}
	defer src.Close()
	preview, err := h.service.PreviewImport(src)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, preview)
}

func (h *ArchiveHandler) Import(c echo.Context) error {
	file, err := c.FormFile("archive")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "archive is required"})
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid archive"})
	}
	defer src.Close()
	actorID, _ := c.Get("user_id").(string)
	subject, err := h.service.Import(c.Request().Context(), src, actorID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, subject)
}

var _ = bytes.NewBuffer
