package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// stubDisciplineService implements the methods used by the handler for testing

type stubDisciplineService struct {
	list    []*ent.Discipline
	get     *ent.Discipline
	err     error
	created *ent.Discipline
	updated *ent.Discipline
	deleted uuid.UUID
}

func (s *stubDisciplineService) List(ctx context.Context) ([]*ent.Discipline, error) {
	return s.list, s.err
}
func (s *stubDisciplineService) GetByID(ctx context.Context, id uuid.UUID) (*ent.Discipline, error) {
	return s.get, s.err
}

func (s *stubDisciplineService) Create(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	s.created = d
	return d, s.err
}
func (s *stubDisciplineService) Update(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	s.updated = d
	return d, s.err
}
func (s *stubDisciplineService) Delete(ctx context.Context, id uuid.UUID) error {
	s.deleted = id
	return s.err
}

func TestDisciplineHandler_List(t *testing.T) {
	sample := &ent.Discipline{ID: uuid.New(), Name: "foo", Slug: "foo", Edges: ent.DisciplineEdges{Country: &ent.Country{ID: uuid.New()}}}
	stub := &stubDisciplineService{list: []*ent.Discipline{sample}}
	h := NewDisciplineHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/disciplines", nil)
	w := httptest.NewRecorder()
	h.ListDisciplines(w, req)
	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	var out []DisciplineResponse
	err := json.NewDecoder(res.Body).Decode(&out)
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "foo", out[0].Name)
}

func TestDisciplineHandler_Create(t *testing.T) {
	stub := &stubDisciplineService{}
	h := NewDisciplineHandler(stub)

	payload := `{"name":"bar","slug":"bar","countryId":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/disciplines", strings.NewReader(payload))
	w := httptest.NewRecorder()
	h.CreateDiscipline(w, req)
	res := w.Result()
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	var out DisciplineResponse
	err := json.NewDecoder(res.Body).Decode(&out)
	assert.NoError(t, err)
	assert.Equal(t, "bar", out.Name)
	assert.NotNil(t, stub.created)
}
