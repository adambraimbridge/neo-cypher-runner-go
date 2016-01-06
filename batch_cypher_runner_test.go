package neocypherrunner

import (
	"errors"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAllQueriesRun(t *testing.T) {
	assert := assert.New(t)
	mr := &mockRunner{}
	batchCypherRunner := NewBatchCypherRunner(mr, 3)

	errCh := make(chan error)

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "First"},
			&neoism.CypherQuery{Statement: "Second"},
		})
	}()

	go func() {
		time.Sleep(time.Millisecond * 1)
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "Third"},
		})
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		assert.NoError(err, "Got an error for %d", i)
	}

	expected := []*neoism.CypherQuery{
		&neoism.CypherQuery{Statement: "First"},
		&neoism.CypherQuery{Statement: "Second"},
		&neoism.CypherQuery{Statement: "Third"},
	}

	assert.Equal(expected, mr.queriesRun, "queries didn't match")
}

func TestQueryBatching(t *testing.T) {
	assert := assert.New(t)

	dr := &delayRunner{make(chan []*neoism.CypherQuery)}
	batchCypherRunner := NewBatchCypherRunner(dr, 3)

	errCh := make(chan error)

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "First"},
		})
	}()

	go func() {
		time.Sleep(time.Millisecond * 1)
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "Second"},
		})
	}()

	go func() {
		time.Sleep(time.Millisecond * 2)
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "Third"},
		})
	}()

	time.Sleep(3 * time.Millisecond)
	// first should have completed, second and third should be queued for next batch

	assert.Equal([]*neoism.CypherQuery{
		&neoism.CypherQuery{Statement: "First"},
	}, <-dr.queriesRun)

	assert.Equal([]*neoism.CypherQuery{
		&neoism.CypherQuery{Statement: "Second"},
		&neoism.CypherQuery{Statement: "Third"},
	}, <-dr.queriesRun)

	for i := 0; i < 3; i++ {
		err := <-errCh
		assert.NoError(err, "Got an error for %d", i)
	}

}

func TestEveryoneGetsErrorOnFailure(t *testing.T) {
	assert := assert.New(t)
	mr := &failRunner{}
	batchCypherRunner := NewBatchCypherRunner(mr, 3)

	errCh := make(chan error)

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "First"},
			&neoism.CypherQuery{Statement: "Second"},
		})
	}()

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "Third"},
		})
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		assert.Error(err, "Didn't get an error for %d", i)
	}

	assert.Equal(len(errCh), 0, "too many errors")
}

type mockRunner struct {
	queriesRun []*neoism.CypherQuery
}

func (mr *mockRunner) CypherBatch(queries []*neoism.CypherQuery) error {
	mr.queriesRun = append(mr.queriesRun, queries...)
	return nil
}

type failRunner struct {
}

func (mr *failRunner) CypherBatch(queries []*neoism.CypherQuery) error {
	return errors.New("UNIT TESTING: Deliberate fail for every query")
}

type delayRunner struct {
	queriesRun chan []*neoism.CypherQuery
}

func (dr *delayRunner) CypherBatch(queries []*neoism.CypherQuery) error {
	dr.queriesRun <- queries
	return nil
}
