package topology_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vitech-team/sdlcctl/cmd/topology"
	"github.com/vitech-team/sdlcctl/cmd/utils"
	"testing"
)

func TestNewTopologyCmd(t *testing.T) {
	options := utils.Options{}
	cmd, _ := topology.NewTopologyCmd(&options)
	options.AddBaseFlags(cmd)
	cmd.SetArgs([]string{
		"--gitUrl", "https://github.com/vitech-team/test-sk-env.git",
		"--hfd", "/Users/serhiykrupka/test-clone",
	})

	//o.HelmfileDir = "/Users/serhiykrupka/test-clone"
	//o.GitUrl = "https://github.com/vitech-team/test-sk-env.git"
	//err := o.Print()
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestNewTopologyCmdTested(t *testing.T) {
	options := utils.Options{}
	cmd, _ := topology.NewTopologyCmd(&options)
	options.AddBaseFlags(cmd)
	cmd.SetArgs([]string{
		"tested",
		"--gitUrl", "https://github.com/vitech-team/test-sk-env.git",
		"--hfd", "/Users/serhiykrupka/test-clone",
		"--status", "ok",
		"--repo", "https://github.com",
		"--commit", "123",
		"--image", "gcr.io",
		"--report", "rep/ttt/",
	})

	//o.HelmfileDir = "/Users/serhiykrupka/test-clone"
	//o.GitUrl = "https://github.com/vitech-team/test-sk-env.git"
	//err := o.Print()
	err := cmd.Execute()
	assert.NoError(t, err)
}
