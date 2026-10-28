package service

import (
	"context"
	"errors"
	"github-watcher/internal/models"
	"github-watcher/internal/storage"
	"testing"
)

func TestParseName(t *testing.T) {
	input := []struct {
		input     string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{"golang/go", "golang", "go", false},
		{"golang", "", "", true},
		{"", "", "", true},
		{"golang/go/golang", "", "", true},
	}

	for _, test := range input {
		owner, name, err := parseName(test.input)
		if err != nil && !test.wantErr {
			t.Errorf("parseName(%q): unexpected error: %v", test.input, err)
		}
		if err == nil && test.wantErr {
			t.Errorf("parseName() error = %v, wantErr %v", err, test.wantErr)
		}
		if owner != test.wantOwner {
			t.Errorf("parseName() error = %v, owner = %v, want %v", err, owner, test.wantOwner)
		}
		if name != test.wantName {
			t.Errorf("parseName() error = %v, name %v, want %v", err, name, test.wantName)
		}
	}
}

type FakeStorage struct {
	deleteCalled bool
	deleteName   string
	deleteErr    error
	listRepos    []models.Repository
	listReposErr error
}

func (f *FakeStorage) DeleteRepository(ctx context.Context, fullName string) error {
	f.deleteCalled = true
	f.deleteName = fullName
	return f.deleteErr
}

func (f *FakeStorage) AddRepository(ctx context.Context, fullName string) (models.Repository, error) {
	return models.Repository{}, f.deleteErr
}

func (f *FakeStorage) ListRepositories(ctx context.Context) ([]models.Repository, error) {
	return f.listRepos, f.listReposErr
}

func (f *FakeStorage) SaveSnapshot(ctx context.Context, snapshot models.RepositorySnapshot) error {
	return nil
}

func (f *FakeStorage) ListSnapshots(ctx context.Context, repositoryID int) ([]models.RepositorySnapshot, error) {
	return []models.RepositorySnapshot{}, f.deleteErr
}

func TestDelete(t *testing.T) {
	fakeStorage := &FakeStorage{}
	service := NewService(nil, fakeStorage, nil)

	err := service.Delete(context.Background(), "golang/go")

	if err != nil {
		t.Errorf("Delete() unexpected error = %v", err)
	}

	if !fakeStorage.deleteCalled {
		t.Error("Delete() was not called")
	}

	if fakeStorage.deleteName != "golang/go" {
		t.Errorf("Delete() deletedName = %q, want %q", fakeStorage.deleteName, "golang/go")
	}
}

func TestDeleteInvalidName(t *testing.T) {
	fakeStorage := &FakeStorage{}
	service := NewService(nil, fakeStorage, nil)

	err := service.Delete(context.Background(), "golang")

	if err == nil {
		t.Error("Delete() expected error, got nil")
	}

	if fakeStorage.deleteCalled {
		t.Error("Delete() called storage with invalid name")
	}
}

func TestDeleteRepositoryNotFound(t *testing.T) {
	fakeStorage := &FakeStorage{
		deleteErr: storage.ErrRepositoryNotFound,
	}
	service := NewService(nil, fakeStorage, nil)

	err := service.Delete(context.Background(), "golang/go")

	if !errors.Is(err, storage.ErrRepositoryNotFound) {
		t.Errorf("Delete() error = %v, want %v", err, storage.ErrRepositoryNotFound)
	}
}

func TestDeleteStorageError(t *testing.T) {
	storageErr := errors.New("database connection failed")
	fakeStorage := &FakeStorage{
		deleteErr: storageErr,
	}
	service := NewService(nil, fakeStorage, nil)

	err := service.Delete(context.Background(), "golang/go")

	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}

	if !errors.Is(err, storageErr) {
		t.Errorf("Delete() error = %v, want wrapped error %v", err, storageErr)
	}
}

func TestList(t *testing.T) {
	fakeStorage := &FakeStorage{
		listRepos: []models.Repository{
			{Name: "go", FullName: "golang/go"},
			{Name: "flecc1", FullName: "flecc1/URL-Shortener"},
		},
	}

	service := NewService(nil, fakeStorage, nil)

	repo, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() unexpected error = %v", err)
	}

	if len(repo) != len(fakeStorage.listRepos) {
		t.Fatalf("List() returned repo length = %v, want %v", len(repo), len(fakeStorage.listRepos))
	}

	for i := range repo {
		if repo[i].Name != fakeStorage.listRepos[i].Name || repo[i].FullName != fakeStorage.listRepos[i].FullName {
			t.Errorf("List() returned repo = %v, want %v", repo, fakeStorage.listRepos)
		}
	}

}

func TestListStorageError(t *testing.T) {
	storageErr := errors.New("database connection failed")

	fakeStorage := &FakeStorage{
		listReposErr: storageErr,
	}

	service := NewService(nil, fakeStorage, nil)
	_, err := service.List(context.Background())
	if err == nil {
		t.Fatal("List() expected error, got nil")
	}

	if !errors.Is(err, storageErr) {
		t.Errorf("List() error = %v, want %v", err, storageErr)
	}
}
