package service

import (
	"context"

	"github.com/macleodmac/pglet/internal/client"
)

// BeginTransaction opens a transaction for the given tab. If one is already
// open it is a no-op, keeping the button idempotent.
func (s *Service) BeginTransaction(ctx context.Context, tabID string) error {
	cl, err := s.requireClient()
	if err != nil {
		return err
	}
	if s.InTransaction(tabID) {
		return nil
	}

	// Not under txMu: waiting for a pooled connection there would block every
	// other tab's commit and rollback, which are what free connections up.
	tx, err := cl.Begin(ctx)
	if err != nil {
		return err
	}
	if !s.storeTx(tabID, tx) {
		tx.Rollback()
	}
	return nil
}

// OpenTransactions lists the tabs that hold an open transaction.
func (s *Service) OpenTransactions() []string {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	ids := make([]string, 0, len(s.txns))
	for id := range s.txns {
		ids = append(ids, id)
	}
	return ids
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
// whenever the client is replaced, so no tab keeps writing to a database the
// UI no longer shows.
func (s *Service) rollbackAllTransactions() {
	s.txMu.Lock()
	txns := s.txns
	s.txns = make(map[string]*client.Tx)
	s.txMu.Unlock()

	for _, tx := range txns {
		tx.Rollback()
	}
}
