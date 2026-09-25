package health

import (
	"context"

	"github.com/rs/zerolog"
)

type DatabaseChecker interface {
	Ping(context.Context) error
}

type Service struct {
	database DatabaseChecker
	log      zerolog.Logger
}

func NewService(database DatabaseChecker, log zerolog.Logger) *Service {
	return &Service{database: database, log: log}
}

func (s *Service) Check(ctx context.Context) error {
	if err := s.database.Ping(ctx); err != nil {
		s.log.Error().Err(err).Msg("health check failed")
		return err
	}
	return nil
}
