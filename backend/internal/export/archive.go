package exporter

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/domain"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

const (
	maxArchiveBytes      = 64 << 20
	maxArchiveFiles      = 256
	maxArchiveEntryBytes = 20 << 20
	maxArchiveExpanded   = 128 << 20
)

type EvidenceManifest struct {
	FileName         string `json:"file_name"`
	OriginalFilename string `json:"original_filename,omitempty"`
	SHA256           string `json:"sha256,omitempty"`
	Type             string `json:"type"`
	Title            string `json:"title,omitempty"`
	Description      string `json:"description,omitempty"`
	URL              string `json:"url,omitempty"`
}
type EventManifest struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	Details        string             `json:"details"`
	Status         string             `json:"status"`
	Severity       int                `json:"severity,omitempty"`
	OccurredFrom   *time.Time         `json:"occurred_from,omitempty"`
	OccurredTo     *time.Time         `json:"occurred_to,omitempty"`
	CorrectionNote *string            `json:"correction_note,omitempty"`
	Evidence       []EvidenceManifest `json:"evidence"`
}
type Manifest struct {
	SchemaVersion int              `json:"schema_version"`
	PublicID      string           `json:"public_id"`
	DisplayName   string           `json:"display_name"`
	Accounts      []models.Account `json:"accounts"`
	Events        []EventManifest  `json:"events"`
	ExportedAt    time.Time        `json:"exported_at"`
}

type ArchiveService struct {
	subjects *repository.SubjectRepository
	events   *repository.EventRepository
	evidence *repository.EvidenceRepository
	store    storage.Storage
	// Reduces same-instance contention; correctness across instances comes from
	// unique operation object keys and database uniqueness, not this mutex.
	mu sync.Mutex
}

func NewArchiveService(s *repository.SubjectRepository, e *repository.EventRepository, v *repository.EvidenceRepository, store storage.Storage) *ArchiveService {
	return &ArchiveService{subjects: s, events: e, evidence: v, store: store}
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 || manifest.PublicID == "" || manifest.DisplayName == "" {
		return fmt.Errorf("unsupported or missing manifest")
	}
	return nil
}

func evidencePrefix(publicID string) string {
	return "subjects/" + publicID + "/evidence/"
}

func validateEvidenceName(publicID, name string) error {
	if name == "" {
		return fmt.Errorf("missing evidence file name")
	}
	cleaned := path.Clean("/" + name)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned != name || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid evidence file name: %s", name)
	}
	prefix := evidencePrefix(publicID)
	if !strings.HasPrefix(name, prefix) {
		return fmt.Errorf("evidence file outside subject namespace: %s", name)
	}
	return nil
}

func readArchive(r io.Reader) (Manifest, map[string][]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxArchiveBytes+1))
	if err != nil {
		return Manifest{}, nil, err
	}
	if len(b) > maxArchiveBytes {
		return Manifest{}, nil, fmt.Errorf("archive exceeds compressed size limit")
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return Manifest{}, nil, err
	}
	if len(zr.File) == 0 || len(zr.File) > maxArchiveFiles {
		return Manifest{}, nil, fmt.Errorf("archive file count out of range")
	}

	files := make(map[string][]byte, len(zr.File))
	var manifest Manifest
	var expanded uint64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.UncompressedSize64 > maxArchiveEntryBytes {
			return Manifest{}, nil, fmt.Errorf("archive entry too large: %s", f.Name)
		}
		if expanded+f.UncompressedSize64 > maxArchiveExpanded {
			return Manifest{}, nil, fmt.Errorf("archive expanded size limit exceeded")
		}
		if _, exists := files[f.Name]; exists {
			return Manifest{}, nil, fmt.Errorf("duplicate archive entry: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return Manifest{}, nil, err
		}
		content, err := io.ReadAll(io.LimitReader(rc, int64(maxArchiveEntryBytes)+1))
		rc.Close()
		if err != nil {
			return Manifest{}, nil, err
		}
		if len(content) > maxArchiveEntryBytes {
			return Manifest{}, nil, fmt.Errorf("archive entry too large: %s", f.Name)
		}
		expanded += uint64(len(content))
		files[f.Name] = content
		if f.Name == "manifest.json" {
			if err := json.Unmarshal(content, &manifest); err != nil {
				return Manifest{}, nil, err
			}
		}
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, nil, err
	}

	seenEvidence := map[string]bool{}
	for _, event := range manifest.Events {
		if event.Severity < 0 || event.Severity > 5 {
			return Manifest{}, nil, fmt.Errorf("invalid event severity")
		}
		if event.OccurredFrom != nil && event.OccurredTo != nil && event.OccurredTo.Before(*event.OccurredFrom) {
			return Manifest{}, nil, fmt.Errorf("invalid event occurrence range")
		}
		switch event.Status {
		case "", "published", "corrected", "withdrawn", "draft", "hidden_pending_review":
		default:
			return Manifest{}, nil, fmt.Errorf("invalid event status: %s", event.Status)
		}
		for _, evidence := range event.Evidence {
			if evidence.Type == "link" {
				if evidence.FileName != "" || evidence.SHA256 != "" || evidence.OriginalFilename != "" {
					return Manifest{}, nil, fmt.Errorf("link evidence must not include body metadata")
				}
				if err := domain.ValidateLinkEvidence(evidence.Title, evidence.URL); err != nil {
					return Manifest{}, nil, fmt.Errorf("invalid link evidence")
				}
				continue
			}
			if evidence.FileName == "" {
				return Manifest{}, nil, fmt.Errorf("missing evidence file name")
			}
			if err := validateEvidenceName(manifest.PublicID, evidence.FileName); err != nil {
				return Manifest{}, nil, err
			}
			if seenEvidence[evidence.FileName] {
				return Manifest{}, nil, fmt.Errorf("duplicate evidence file name: %s", evidence.FileName)
			}
			seenEvidence[evidence.FileName] = true
			if evidence.Type != "file" && evidence.Type != "text" {
				return Manifest{}, nil, fmt.Errorf("invalid evidence type: %s", evidence.Type)
			}
			if len(evidence.SHA256) != 64 {
				return Manifest{}, nil, fmt.Errorf("missing evidence hash: %s", evidence.FileName)
			}
			content, ok := files[evidence.FileName]
			if !ok {
				return Manifest{}, nil, fmt.Errorf("missing evidence file: %s", evidence.FileName)
			}
			sum := sha256.Sum256(content)
			if evidence.SHA256 != hex.EncodeToString(sum[:]) {
				return Manifest{}, nil, fmt.Errorf("evidence hash mismatch: %s", evidence.FileName)
			}
		}
	}
	for name := range files {
		switch name {
		case "manifest.json", "README.txt":
			continue
		default:
			if !seenEvidence[name] {
				return Manifest{}, nil, fmt.Errorf("unreferenced archive entry: %s", name)
			}
		}
	}
	return manifest, files, nil
}

func eventManifest(event models.Event) EventManifest {
	return EventManifest{ID: event.ID, Title: event.Title, Details: event.Details, Status: event.Status,
		Severity: event.Severity, OccurredFrom: event.OccurredFrom, OccurredTo: event.OccurredTo, CorrectionNote: event.CorrectionNote}
}

func importedEvent(m EventManifest, actorID string) models.Event {
	status := m.Status
	if status == "" {
		status = "published"
	}
	severity := m.Severity
	if severity == 0 {
		severity = 1
	} // v1 archives before optional fields
	return models.Event{Title: m.Title, Details: m.Details, Status: status, Severity: severity,
		OccurredFrom: m.OccurredFrom, OccurredTo: m.OccurredTo, CorrectionNote: m.CorrectionNote, SubmittedBy: &actorID}
}

func (s *ArchiveService) Build(ctx context.Context, publicID string) ([]byte, error) {
	subject, err := s.subjects.GetSubjectByID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	events, err := s.events.ListBySubject(ctx, subject.ID)
	if err != nil {
		return nil, err
	}
	m := Manifest{SchemaVersion: 1, PublicID: subject.PublicID, DisplayName: subject.DisplayName, Accounts: subject.Accounts, ExportedAt: time.Now().UTC()}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	entryCount, expanded := 2, 0 // manifest + README
	for _, event := range events {
		em := eventManifest(event)
		items, err := s.evidence.GetEvidenceByEventID(ctx, event.ID)
		if err != nil {
			return nil, err
		}
		for _, v := range items {
			item := EvidenceManifest{Type: v.Type}
			if v.Title != nil {
				item.Title = *v.Title
			}
			if v.Description != nil {
				item.Description = *v.Description
			}
			if v.Type == "link" {
				if v.URL != nil {
					item.URL = *v.URL
				}
				em.Evidence = append(em.Evidence, item)
				continue
			}
			if v.URL != nil {
				item.URL = *v.URL
			}
			if v.OriginalFilename != nil {
				item.OriginalFilename = *v.OriginalFilename
			}
			if v.SHA256 != nil {
				item.SHA256 = *v.SHA256
			}
			if v.StorageKey != nil {
				item.FileName = *v.StorageKey
				rc, err := s.store.Open(ctx, *v.StorageKey)
				if err != nil {
					return nil, fmt.Errorf("open evidence %s: %w", *v.StorageKey, err)
				}
				b, err := io.ReadAll(io.LimitReader(rc, maxArchiveEntryBytes+1))
				rc.Close()
				if len(b) > maxArchiveEntryBytes {
					return nil, fmt.Errorf("archive entry too large: %s", *v.StorageKey)
				}
				if err != nil {
					return nil, fmt.Errorf("read evidence %s: %w", *v.StorageKey, err)
				}
				sum := sha256.Sum256(b)
				if item.SHA256 != "" && item.SHA256 != hex.EncodeToString(sum[:]) {
					return nil, fmt.Errorf("evidence hash mismatch: %s", *v.StorageKey)
				}
				if item.SHA256 == "" {
					item.SHA256 = hex.EncodeToString(sum[:])
				}
				entryCount++
				expanded += len(b)
				if entryCount > maxArchiveFiles || expanded > maxArchiveExpanded {
					return nil, fmt.Errorf("archive resource limits exceeded")
				}
				w, err := zw.Create(*v.StorageKey)
				if err != nil {
					return nil, err
				}
				if _, err := w.Write(b); err != nil {
					return nil, err
				}
			}
			em.Evidence = append(em.Evidence, item)
		}
		m.Events = append(m.Events, em)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	w, err := zw.Create("manifest.json")
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	readme := "UniBlack 对象归档包\n\nmanifest.json 使用 schema_version 1。evidence/ 中的文件由 manifest 的 SHA-256 校验。文本证据为 UTF-8 txt。导入前必须预览，不能覆盖已存在的 public_id。\n"
	w, err = zw.Create("README.txt")
	if err != nil {
		return nil, err
	}
	if _, err := w.Write([]byte(readme)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if out.Len() > maxArchiveBytes || expanded+len(data)+len(readme) > maxArchiveExpanded || len(data) > maxArchiveEntryBytes {
		return nil, fmt.Errorf("archive resource limits exceeded")
	}
	return out.Bytes(), nil
}

type ImportPreview struct {
	PublicID  string   `json:"public_id"`
	Conflicts []string `json:"conflicts"`
	Valid     bool     `json:"valid"`
}

func (s *ArchiveService) PreviewImport(r io.Reader) (ImportPreview, error) {
	manifest, _, err := readArchive(r)
	if err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{PublicID: manifest.PublicID, Valid: true}
	if _, err := s.subjects.GetSubjectByID(context.Background(), manifest.PublicID); err == nil {
		preview.Conflicts = []string{manifest.PublicID}
		preview.Valid = false
	} else if !errors.Is(err, repository.ErrSubjectNotFound) {
		return ImportPreview{}, err
	}
	accountConflicts, err := s.subjects.AccountConflicts(context.Background(), manifest.Accounts)
	if err != nil {
		return ImportPreview{}, err
	}
	if len(accountConflicts) > 0 {
		preview.Conflicts = append(preview.Conflicts, accountConflicts...)
		preview.Valid = false
	}
	return preview, nil
}

// Import writes a validated archive only if its public ID is still absent.
// Files are stored under the subject namespace before metadata and removed if the
// transaction fails. Each import remaps physical object keys to its own random
// operation namespace; database uniqueness decides metadata conflicts.
func (s *ArchiveService) Import(ctx context.Context, r io.Reader, actorID string, source string) (*models.Subject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	manifest, files, err := readArchive(r)
	if err != nil {
		return nil, err
	}
	if _, err := s.subjects.GetSubjectByID(ctx, manifest.PublicID); err == nil {
		return nil, fmt.Errorf("subject public ID already exists")
	} else if !errors.Is(err, repository.ErrSubjectNotFound) {
		return nil, err
	}
	accountConflicts, err := s.subjects.AccountConflicts(ctx, manifest.Accounts)
	if err != nil {
		return nil, err
	}
	if len(accountConflicts) > 0 {
		return nil, fmt.Errorf("account identity already exists")
	}

	// Per-operation physical identities prevent a losing instance from
	// overwriting or compensating a successful import's objects.
	var operation [16]byte
	if _, err := rand.Read(operation[:]); err != nil {
		return nil, err
	}
	prefix := evidencePrefix(manifest.PublicID) + "imports/" + hex.EncodeToString(operation[:]) + "/"
	physicalKeys := map[string]string{}
	stored := make([]string, 0)
	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		for _, key := range stored {
			_ = s.store.Delete(cleanupCtx, key)
		}
	}

	for _, event := range manifest.Events {
		for _, evidence := range event.Evidence {
			if evidence.FileName == "" || evidence.Type == "link" {
				continue
			}
			if err := validateEvidenceName(manifest.PublicID, evidence.FileName); err != nil {
				cleanup()
				return nil, err
			}
			key := prefix + fmt.Sprintf("%03d_%s", len(stored)+1, path.Base(evidence.FileName))
			physicalKeys[evidence.FileName] = key
			if _, err := s.store.Upload(ctx, key, bytes.NewReader(files[evidence.FileName]), "application/octet-stream"); err != nil {
				cleanup()
				return nil, err
			}
			stored = append(stored, key)
		}
	}

	subject := &models.Subject{PublicID: manifest.PublicID, DisplayName: manifest.DisplayName, Status: "active", CreatedBy: &actorID}
	for i := range manifest.Accounts {
		if manifest.Accounts[i].CustomAttributes == nil {
			manifest.Accounts[i].CustomAttributes = map[string]interface{}{}
		}
		manifest.Accounts[i].ID = ""
		manifest.Accounts[i].SubjectID = ""
	}
	events := make([]models.Event, 0, len(manifest.Events))
	evidenceRows := make([]repository.EventEvidence, 0)
	for eventIndex, m := range manifest.Events {
		events = append(events, importedEvent(m, actorID))
		for _, item := range m.Evidence {
			if item.Type == "link" {
				title, description, url := item.Title, item.Description, item.URL
				evidenceRows = append(evidenceRows, repository.EventEvidence{EventIndex: eventIndex, Evidence: models.Evidence{Type: "link", Title: &title, Description: &description, URL: &url, UploadedBy: &actorID}})
				continue
			}
			if item.FileName == "" {
				continue
			}
			key := physicalKeys[item.FileName]
			original := item.OriginalFilename
			hash := item.SHA256
			size := int64(len(files[item.FileName]))
			mime := "application/octet-stream"
			if item.Type == "text" {
				mime = "text/plain"
			}
			evidenceRows = append(evidenceRows, repository.EventEvidence{
				EventIndex: eventIndex,
				Evidence: models.Evidence{
					Type:             item.Type,
					Title:            &item.Title,
					Description:      &item.Description,
					StorageKey:       &key,
					OriginalFilename: &original,
					SHA256:           &hash,
					FileSize:         &size,
					MimeType:         &mime,
					UploadedBy:       &actorID,
				},
			})
		}
	}
	changes := map[string]interface{}{"public_id": manifest.PublicID}
	if source != "" {
		changes["import_source"] = source
	}
	audit := &models.AuditLog{UserID: &actorID, Action: "import", ResourceType: "subject", Changes: changes}
	if err := s.events.PublishWithEvidence(ctx, subject, manifest.Accounts, events, evidenceRows, audit); err != nil {
		cleanup()
		return nil, err
	}
	return subject, nil
}
