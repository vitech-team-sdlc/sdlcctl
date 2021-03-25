package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRootCmd(t *testing.T) {
	cmd := Main()
	cmd.SetArgs([]string{
		"promotion", "valid",
		"--gitUrl", "https://github.com/vitech-team/test-sk-env.git",
		"--hfd", "/Users/serhiykrupka/test-clone",
	})

	//o.HelmfileDir = "/Users/serhiykrupka/test-clone"
	//o.GitUrl = "https://github.com/vitech-team/test-sk-env.git"
	//err := o.Print()
	err := cmd.Execute()
	assert.NoError(t, err)
}
