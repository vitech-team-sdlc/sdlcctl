package utils

import (
	jxClient "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	sdlcClient "github.com/vitech-team/sdlcctl/client/clientset/versioned"
	k8s "k8s.io/client-go/kubernetes"
)

type Options struct {
	Helmfile    string
	HelmfileDir string
	GitUrl      string

	JxClient   jxClient.Interface
	LtClient   sdlcClient.Interface
	KubeClient k8s.Interface
}

func (options *Options) AddBaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&options.Helmfile,
		"helmfile",
		"f",
		"helmfile.yaml",
		"filter by specific environment",
	)

	cmd.PersistentFlags().StringVarP(
		&options.HelmfileDir,
		"hfd",
		"d",
		"./",
		"HelmFiles root directory",
	)

	cmd.PersistentFlags().StringVarP(
		&options.GitUrl,
		"gitUrl",
		"g",
		"",
		"Git url where helmfiles stored",
	)

	if err := cmd.MarkPersistentFlagDirname("hfd"); err != nil {
		panic(err.Error())
	}

	if err := cmd.MarkPersistentFlagRequired("gitUrl"); err != nil {
		panic(err.Error())
	}

	if err := cmd.MarkPersistentFlagFilename("helmfile"); err != nil {
		panic(err.Error())
	}
}
