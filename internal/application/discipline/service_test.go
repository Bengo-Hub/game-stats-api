package discipline

import (
	"context"
	"errors"
	"testing"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// stubRepo implements Repository for tests and captures calls.

type stubRepo struct {
	created *ent.Discipline
	updated *ent.Discipline
	deleted uuid.UUID
	list    []*ent.Discipline
	find    *ent.Discipline
	err     error
}

func (s *stubRepo) Create(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.created = d
	return d, nil
}
func (s *stubRepo) GetByID(ctx context.Context, id uuid.UUID) (*ent.Discipline, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.find, nil
}
func (s *stubRepo) GetBySlug(ctx context.Context, slug string) (*ent.Discipline, error) {
	return nil, nil
}
func (s *stubRepo) List(ctx context.Context) ([]*ent.Discipline, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.list, nil
}
func (s *stubRepo) Update(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.updated = d
	return d, nil
}
func (s *stubRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if s.err != nil {
		return s.err
	}
	s.deleted = id
	return nil
}

func TestService_List(t *testing.T) {
	expected := []*ent.Discipline{{Name: "foo"}}
	srv := NewService(&stubRepo{list: expected})
	got, err := srv.List(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_Create_Update_Delete(t *testing.T) {
	id := uuid.Must(uuid.NewRandom())
	d := &ent.Discipline{ID: id, Name: "bar"}
	stub := &stubRepo{}
	srv := NewService(stub)

	created, err := srv.Create(context.Background(), d)
	assert.NoError(t, err)
	assert.Equal(t, d, created)
	assert.Equal(t, d, stub.created)

	stub.find = d
	f, err := srv.GetByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, d, f)

	d.Name = "baz"
	updated, err := srv.Update(context.Background(), d)
	assert.NoError(t, err)
	assert.Equal(t, d, updated)
	assert.Equal(t, d, stub.updated)

	err = srv.Delete(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, id, stub.deleted)
}

func TestService_ErrorPropagation(t *testing.T) {
	stub := &stubRepo{err: errors.New("boom")}
	srv := NewService(stub)
	_, err := srv.List(context.Background())
	assert.Error(t, err)
	_, err = srv.Create(context.Background(), &ent.Discipline{})
	assert.Error(t, err)
	_, err = srv.Update(context.Background(), &ent.Discipline{})
	assert.Error(t, err)
	err = srv.Delete(context.Background(), uuid.New())
	assert.Error(t, err)
}
