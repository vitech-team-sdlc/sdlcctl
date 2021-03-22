package promotion

import (
	"github.com/stretchr/testify/assert"
	"github.com/vitech-team/sdlcctl/cmd/utils"
	"testing"
)

func TestNewTopologyCmd(t *testing.T) {
	options := utils.Options{}
	cmd, _ := NewPromotionCmd(&options)

	options.AddBaseFlags(cmd)
	cmd.SetArgs([]string{
		"valid",
		"--gitUrl", "https://github.com/vitech-team/test-sk-env.git",
		"--hfd", "/Users/serhiykrupka/test-clone",
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}
