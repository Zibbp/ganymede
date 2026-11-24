package admin

import (
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/storage"
)

type Service struct {
	Store          *database.Database
	StorageChecker storage.Checker
}

func NewService(store *database.Database, storageChecker storage.Checker) *Service {
	return &Service{Store: store, StorageChecker: storageChecker}
}
