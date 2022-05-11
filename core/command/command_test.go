package command_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/core/command"
)

var (
	_ command.Command = commandTest1{}
	_ command.Command = commandTest2{}
)

type commandTest1 struct{}

func (commandTest1) Name() string { return "command_test_1" }

type commandTest2 struct{}

func (commandTest2) Name() string { return "command_test_2" }

func TestGenericEnvelope(t *testing.T) {
	cmd1 := command.ToEnvelope(commandTest1{})
	genericCmd1 := cmd1.ToGenericEnvelope()

	v1, ok := command.FromGenericEnvelope[commandTest1](genericCmd1)
	assert.Equal(t, cmd1, v1)
	assert.True(t, ok)

	v2, ok := command.FromGenericEnvelope[commandTest2](genericCmd1)
	assert.Zero(t, v2)
	assert.False(t, ok)
}
