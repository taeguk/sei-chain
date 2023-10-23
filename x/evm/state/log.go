package state

import (
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DBImpl) AddLog(l *ethtypes.Log) {
	err := s.k.AddLog(s.ctx, l)
	s.err = err
}

func (s *DBImpl) GetLogs() ([]*ethtypes.Log, error) {
	return s.k.GetLogs(s.ctx)
}
