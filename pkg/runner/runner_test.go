package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorsBeforeQuitting0(t *testing.T) {
	m := &mockedRunner{}
	m.hits = []MockedHit{
		MockedHit{Error: errors.New("error1")},
		MockedHit{Error: errors.New("error2")},
		MockedHit{},
	}
	url := "errors-before-quitting-0"

	loopHits, err := Loop(m, url)

	assert.Equal(t, 1, len(loopHits.List))
	assert.NotNil(t, err)
	assert.Equal(t, "error1", err.Error())
}

func TestErrorsBeforeQuitting1(t *testing.T) {
	m := &mockedRunner{}
	m.hits = []MockedHit{
		MockedHit{Error: errors.New("error1")},
		MockedHit{Error: errors.New("error2")},
		MockedHit{},
	}
	m.errorsBeforeQuitting = 1
	url := "errors-before-quitting-1"

	loopHits, err := Loop(m, url)

	assert.Equal(t, 2, len(loopHits.List))
	assert.NotNil(t, err)
	assert.Equal(t, "error2", err.Error())
}

func TestErrorsBeforeQuitting2(t *testing.T) {
	m := &mockedRunner{}
	m.hits = []MockedHit{
		MockedHit{Error: errors.New("error1")},
		MockedHit{Error: errors.New("error2")},
		MockedHit{},
	}
	m.errorsBeforeQuitting = 2
	url := "errors-before-quitting-2"

	loopHits, err := Loop(m, url)

	assert.Equal(t, 3, len(loopHits.List))
	assert.Nil(t, err)
}

func TestMaxHitsPerLoop0(t *testing.T) {
	m := &mockedRunner{}
	m.hits = []MockedHit{
		MockedHit{MoreDeferred: true},
		MockedHit{MoreDeferred: true},
		MockedHit{},
	}
	url := "max-hits-per-loop-0"

	loopHits, _ := Loop(m, url)

	assert.Equal(t, 3, len(loopHits.List))
}

func TestMaxHitsPerLoop1(t *testing.T) {
	m := &mockedRunner{}
	m.hits = []MockedHit{
		MockedHit{MoreDeferred: true},
		MockedHit{MoreDeferred: true},
		MockedHit{},
	}
	m.maxHitsPerLoop = 1
	url := "max-hits-per-loop-1"

	loopHits, _ := Loop(m, url)

	assert.Equal(t, 1, len(loopHits.List))
}

func TestMaxHitsPerLoop2(t *testing.T) {
	m := &mockedRunner{}
	m.hits = []MockedHit{
		MockedHit{MoreDeferred: true},
		MockedHit{MoreDeferred: true},
		MockedHit{},
	}
	m.maxHitsPerLoop = 2
	url := "max-hits-per-loop-2"

	loopHits, _ := Loop(m, url)

	assert.Equal(t, 2, len(loopHits.List))
}
