package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Exercise the real handler -> service -> repository path without opening a
// database connection. Capture GORM's final pagination, not just the JSON echo.
func TestListHandlersEchoEffectivePagination(t *testing.T) {
	for _, handler := range []struct {
		name string
		new  func(*gorm.DB) echo.HandlerFunc
	}{
		{"subject", func(db *gorm.DB) echo.HandlerFunc {
			return NewSubjectHandler(service.NewSubjectService(repository.NewSubjectRepository(db))).ListSubjects
		}},
		{"public", func(db *gorm.DB) echo.HandlerFunc {
			return NewPublicAPIHandler(service.NewSubjectService(repository.NewSubjectRepository(db)), nil, nil).ListSubjects
		}},
		{"appeal", func(db *gorm.DB) echo.HandlerFunc {
			return NewAppealHandler(service.NewAppealService(repository.NewAppealRepository(db), nil, nil, nil)).ListAppeals
		}},
		{"submission", func(db *gorm.DB) echo.HandlerFunc {
			return NewSubmissionHandler(service.NewSubmissionService(repository.NewSubmissionRepository(db), nil, nil, nil)).ListSubmissions
		}},
		{"sanction", func(db *gorm.DB) echo.HandlerFunc {
			return NewSanctionHandler(service.NewSanctionService(repository.NewSanctionRepository(db), nil)).List
		}},
		{"user", func(db *gorm.DB) echo.HandlerFunc {
			return NewUserManagementHandler(db).ListUsers
		}},
		{"case", func(db *gorm.DB) echo.HandlerFunc {
			return NewCaseHandler(service.NewCaseService(repository.NewCaseRepository(db), nil, nil, nil)).ListCases
		}},
		{"access list", func(db *gorm.DB) echo.HandlerFunc {
			return NewSystemSettingHandler(service.NewSystemSettingService(nil, repository.NewAccessListRepository(db), nil)).ListAccessListEntries
		}},
	} {
		t.Run(handler.name, func(t *testing.T) {
			for _, tc := range []struct {
				name, query string
				page, size  int
			}{
				{"missing", "", 1, 20},
				{"malformed", "?page=bad&page_size=bad", 1, 20},
				{"zero", "?page=0&page_size=0", 1, 20},
				{"negative", "?page=-5&page_size=-10", 1, 20},
				{"oversized", "?page=3&page_size=101", 3, 20},
				{"minimum", "?page=2&page_size=1", 2, 1},
				{"maximum", "?page=2&page_size=100", 2, 100},
				{"valid", "?page=4&page_size=35", 4, 35},
			} {
				t.Run(tc.name, func(t *testing.T) {
					db, err := gorm.Open(postgres.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
					if err != nil {
						t.Fatal(err)
					}
					var limits []clause.Limit
					if err := db.Callback().Query().After("gorm:query").Register("test:capture_pagination", func(tx *gorm.DB) {
						if limit, ok := tx.Statement.Clauses["LIMIT"].Expression.(clause.Limit); ok {
							limits = append(limits, limit)
						}
					}); err != nil {
						t.Fatal(err)
					}
					rec := httptest.NewRecorder()
					c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/list"+tc.query, nil), rec)
					if err := handler.new(db)(c); err != nil {
						t.Fatal(err)
					}
					if rec.Code != http.StatusOK {
						t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
					}
					var response struct {
						Page     int `json:"page"`
						PageSize int `json:"page_size"`
					}
					if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
						t.Fatal(err)
					}
					if response.Page != tc.page || response.PageSize != tc.size {
						t.Fatalf("metadata = (%d, %d), want (%d, %d)", response.Page, response.PageSize, tc.page, tc.size)
					}
					if len(limits) != 1 || limits[0].Limit == nil || *limits[0].Limit != tc.size || limits[0].Offset != (tc.page-1)*tc.size {
						t.Fatalf("query pagination = %#v, want offset %d, limit %d", limits, (tc.page-1)*tc.size, tc.size)
					}
				})
			}
		})
	}
}
