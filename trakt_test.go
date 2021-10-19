package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	e = Episode{
		Season: 1,
		Number: 2,
		Title:  "test ep",
	}

	s = Show{
		Title: "test show",
	}

	m = Movie{
		Title: "test movie",
		Year:  2020,
	}

	ee = Entry{
		Type:    "episode",
		Episode: e,
		Show:    s,
	}

	em = Entry{
		Type:  "movie",
		Movie: m,
	}

	j = traktJSON{ee, em}
)

func TestJson(t *testing.T) {
	assert.Equal(t, j.Latest(), ee.String())
}

func TestString(t *testing.T) {
	cases := []struct {
		name     string
		obj      fmt.Stringer
		expected string
	}{
		{
			name:     "Episode entry",
			obj:      ee,
			expected: "test show 01x02 - test ep",
		},
		{
			name:     "Movie entry",
			obj:      em,
			expected: "test movie (2020)",
		},
		{
			name:     "Episode",
			obj:      e,
			expected: "01x02 - test ep",
		},
		{
			name:     "Show",
			obj:      s,
			expected: "test show",
		},
		{
			name:     "Movie",
			obj:      m,
			expected: "test movie (2020)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.obj.String(), tc.expected)
		})
	}
}
