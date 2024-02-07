package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_initScheme(t *testing.T) {
	scheme := initScheme()
	require.NotNil(t, scheme)
}
