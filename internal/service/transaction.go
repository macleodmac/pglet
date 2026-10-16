package service

import (
	"context"

	"github.com/macleodmac/pglet/internal/client"
)

// BeginTransaction opens a transaction for the given tab. If one is already
// open it is a no-op, keeping the button idempotent.
func (s *Service) BeginTransaction(tabID string) error {
	cl, err := s.requireClient()
	if err != nil {
		return err
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()
	if _, ok := s.txns[tabID]; ok {
		return nil
	}

	tx, err := cl.Begin(context.Background())
	if err != nil {
		return err
	}
	s.txns[tabID] = tx
	return nil
}

// CommitTransaction commits and clears the tab's transaction.
func (s *Service) CommitTransaction(tabID string) error {
	tx, ok := s.takeTx(tabID)
	if !ok {
		return ErrNoTransaction
	}
	return tx.Commit()
}

// RollbackTransaction rolls back and clears the tab's transaction.
func (s *Service) RollbackTransaction(tabID string) error {
	tx, ok := s.takeTx(tabID)
	if !ok {
		return ErrNoTransaction
	}
	return tx.Rollback()
}

// InTransaction reports whether the tab currently has an open transaction.
func (s *Service) InTransaction(tabID string) bool {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	_, ok := s.txns[tabID]
	return ok
}

// getTx returns the tab's transaction without removing it.
func (s *Service) getTx(tabID string) (*client.Tx, bool) {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	tx, ok := s.txns[tabID]
	return tx, ok
}

// storeTx makes tx the tab's open transaction, unless the tab already has one.
func (s *Service) storeTx(tabID string, tx *client.Tx) bool {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	if _, ok := s.txns[tabID]; ok {
		return false
	}
	s.txns[tabID] = tx
	return true
}

// takeTx removes and returns the tab's transaction.
func (s *Service) takeTx(tabID string) (*client.Tx, bool) {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	tx, ok := s.txns[tabID]
	if ok {
		delete(s.txns, tabID)
	}
	return tx, ok
}

// rollbackAllTransactions rolls back and clears every open transaction. Called
// when the underlying connection is torn down (disconnect or database switch)
// so pinned connections are not leaked.
func (s *Service) rollbackAllTransactions() {
	s.txMu.Lock()
	txns := s.txns
	s.txns = make(map[string]*client.Tx)
	s.txMu.Unlock()

	for _, tx := range txns {
		tx.Rollback()
	}
}
