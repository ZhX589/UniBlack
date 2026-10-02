package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/service"
)

func eventVisibleToRequester(c echo.Context, event *models.Event) bool {
	userID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	return service.EventVisibleTo(event, userID, roles)
}

// Filter only embedded events, without changing subject lifecycle or legacy
// Case semantics. Copy the response so filtering cannot mutate service results.
func subjectWithVisibleEvents(c echo.Context, subject *models.Subject) *models.Subject {
	if subject == nil {
		return nil
	}
	result := *subject
	result.Events = nil
	for i := range subject.Events {
		if eventVisibleToRequester(c, &subject.Events[i]) {
			result.Events = append(result.Events, subject.Events[i])
		}
	}
	return &result
}

func subjectsWithVisibleEvents(c echo.Context, subjects []models.Subject) []models.Subject {
	if subjects == nil {
		return nil
	}
	result := make([]models.Subject, len(subjects))
	for i := range subjects {
		result[i] = *subjectWithVisibleEvents(c, &subjects[i])
	}
	return result
}
