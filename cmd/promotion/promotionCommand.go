package promotion

import (
	"context"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	sdlc "github.com/vitech-team/sdlcctl/apis/largetest/v1beta1"
	"github.com/vitech-team/sdlcctl/cmd/topology"
	"github.com/vitech-team/sdlcctl/cmd/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sort"
)

type PromotionOptions struct {
	*utils.Options
}

func NewPromotionCmd(rootOpts *utils.Options) (*cobra.Command, *PromotionOptions) {
	options := &PromotionOptions{rootOpts}

	command := &cobra.Command{
		Use:     "promotion",
		Short:   "prom",
		Example: "check if ",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				panic(err.Error())
			}
		},
	}

	validate := &cobra.Command{
		Use:     "valid",
		Long:    "Check if changes currently made was tested on previous environment",
		Example: "sdlc promotion valid",
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Validate()
			if err != nil {
				os.Exit(1)
			}
		},
	}

	command.AddCommand(validate)

	return command, options
}

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
}

func (opt *PromotionOptions) Validate() error {

	optionsTopology := topology.OptionsTopology{Options: opt.Options}

	environments, err := optionsTopology.GetComparedTopology()

	if err != nil {
		filteredEnvs := findEnvWithPromotion(environments)
		testedEnvs := collectTestExecutions(filteredEnvs, opt)
		for _, env := range testedEnvs {
			if !env.Tested {
				log.WithField("env", env.Name).Info("tested")
			} else {
				log.WithField("env", env.Name).Error("no large test executions found")
				return err
			}
		}
		log.Info("all changes are tested")
	}

	return err
}

func collectTestExecutions(filteredEnvs []utils.Environment, opt *PromotionOptions) []utils.Environment {
	var testedEnvs []utils.Environment
	for i, env := range filteredEnvs {
		if i > 0 {
			previousEnvironment := filteredEnvs[i-1]
			largeTests, err := opt.GetLargeTestExecutions(previousEnvironment)
			if err != nil {
				log.WithField("env", env.Name).WithError(err).Error("can't check large tests")
			}
			matchedLargeTest := findLargeTestExecution(env, largeTests)
			env.Tested = len(matchedLargeTest) > 0
			testedEnvs = append(testedEnvs, env)
		}
	}
	return testedEnvs
}

func (opt *PromotionOptions) GetLargeTestExecutions(env utils.Environment) (*sdlc.LargeTestExecutionList, error) {
	opt.KubeClient, opt.JxClient, opt.LtClient = utils.NewLazyClients(opt.KubeClient, opt.JxClient, opt.LtClient)
	largeTestRuns, err := opt.LtClient.LargetestV1beta1().LargeTestExecutions(env.Spec.Namespace).List(
		context.TODO(), metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}
	return largeTestRuns, err
}

func findLargeTestExecution(env utils.Environment, largeTests *sdlc.LargeTestExecutionList) []sdlc.LargeTestExecution {
	var results []sdlc.LargeTestExecution
	for _, lte := range largeTests.Items {
		if matched(env.Topology, lte.Spec.Topology) {
			results = append(results, lte)
		}
	}
	return results
}

func matched(top1 []sdlc.AppVersion, top2 []sdlc.AppVersion) bool {
	for _, e1 := range top1 {
		if !utils.ContainsVersion(e1, top2) {
			return false
		}
	}
	return true
}

func findEnvWithPromotion(envs []utils.Environment) []utils.Environment {

	var filtered []utils.Environment
	for _, env := range envs {
		if env.Spec.PromotionStrategy != "" && env.Spec.PromotionStrategy != v1.PromotionStrategyTypeNever {
			filtered = append(filtered, env)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return i > j
	})

	return filtered
}
