package topology

import (
	"context"
	"fmt"
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/yaml2s"
	"github.com/olekukonko/tablewriter"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	sdlc "github.com/vitech-team/sdlcctl/apis/largetest/v1beta1"
	sdlcUtils "github.com/vitech-team/sdlcctl/cmd/utils"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sort"
	"strings"
	"time"
)

type OptionsTopology struct {
	*sdlcUtils.Options
}

type OptionsTopologyTested struct {
	Status string
	Report string
	Commit string
	Repo   string
	Image  string
	*OptionsTopology
}

var commandRunner cmdrunner.CommandRunner
var gitClient gitclient.Interface

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
}

func NewTopologyCmd(opts *sdlcUtils.Options) (*cobra.Command, *OptionsTopology) {
	options := &OptionsTopology{opts}
	optionTested := &OptionsTopologyTested{OptionsTopology: options}

	command := &cobra.Command{
		Use:     "topology",
		Example: "bla bla bla",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	printCmd := &cobra.Command{
		Use:     "print",
		Example: "print current topology VS previous",
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Print()
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		},
	}

	testedCmd := &cobra.Command{
		Use:     "tested",
		Example: "create large test execution for current topology ",
		Run: func(cmd *cobra.Command, args []string) {
			err := optionTested.MarkWithLargeTestExec()
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		},
	}

	testedCmd.Flags().StringVarP(
		&optionTested.Status, "status", "", "", "large test status success/failed",
	)
	testedCmd.Flags().StringVarP(
		&optionTested.Report, "report", "", "", "report url or place where it can be found",
	)
	testedCmd.Flags().StringVarP(
		&optionTested.Commit, "commit", "", "", "commit what been tested",
	)
	testedCmd.Flags().StringVarP(
		&optionTested.Repo, "repo", "", "", "repository where version been changed",
	)
	testedCmd.Flags().StringVarP(
		&optionTested.Image, "image", "", "", "large reports produced",
	)

	testedCmd.MarkFlagRequired("status")
	testedCmd.MarkFlagRequired("report")
	testedCmd.MarkFlagRequired("commit")
	testedCmd.MarkFlagRequired("repo")
	testedCmd.MarkFlagRequired("image")

	command.AddCommand(printCmd)
	command.AddCommand(testedCmd)

	return command, options
}

func (opt *OptionsTopologyTested) MarkWithLargeTestExec() error {
	currentHelmState := opt.GetEnvironmentsFromHelmFile(opt.Helmfile, opt.HelmfileDir)
	for _, env := range currentHelmState {
		opt.KubeClient, opt.JxClient, opt.LtClient = sdlcUtils.NewLazyClients(opt.KubeClient, opt.JxClient, opt.LtClient)
		err := createIfNotExists(opt, env)
		if err != nil {
			return err
		}

		lte := &sdlc.LargeTestExecution{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    env.Spec.Namespace,
				GenerateName: fmt.Sprintf("%s-%s", env.Spec.Namespace, opt.Commit),
			},
			Spec: sdlc.LargeTestExecutionSpec{
				Image:       opt.Image,
				Result:      opt.Status,
				Environment: env.Name,
				Namespace:   env.Spec.Namespace,
				Report:      opt.Report,
				Time:        time.Now().String(),
				Topology:    env.Topology,
			},
		}
		created, err := opt.LtClient.LargetestV1beta1().LargeTestExecutions(env.Spec.Namespace).Create(
			context.TODO(), lte, metav1.CreateOptions{},
		)
		if err != nil {
			return err
		}
		log.WithField("name", created.Name).
			WithField("ns", env.Spec.Namespace).
			Info("new LargeTestExecution created")
	}

	return nil
}

func createIfNotExists(opt *OptionsTopologyTested, env sdlcUtils.Environment) error {
	var _, err = opt.KubeClient.CoreV1().Namespaces().Get(context.TODO(), env.Spec.Namespace, metav1.GetOptions{})
	if err != nil {
		if err.(*errors.StatusError).ErrStatus.Reason == metav1.StatusReasonNotFound {
			_, err = opt.KubeClient.CoreV1().Namespaces().Create(
				context.TODO(),
				&v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: env.Spec.Namespace,
					},
				}, metav1.CreateOptions{},
			)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return err
}

func (opt *OptionsTopology) Print() error {

	comparedEnvironments, err := opt.GetComparedTopology()

	if err != nil {
		panic(err.Error())
	}
	var data = [][]string{}

	for _, env := range comparedEnvironments {
		envName := env.Name

		if env.Changed {
			sortByName(env.Topology)
			sortByName(env.PreviousTopology)
			cols := []string{
				envName,
				fmt.Sprintf("%t", env.Changed),
				strings.Join(getNames(env.Topology), ", "),
				strings.Join(getNames(env.PreviousTopology), ", "),
			}
			data = append(data, cols)
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Env", "Env Changed", "Now", "Was"})

	table.AppendBulk(data)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Render()

	return err
}

func getNames(apps []sdlc.AppVersion) []string {
	var names []string
	for _, app := range apps {
		names = append(names, fmt.Sprintf("%s:%s", app.Name, app.Version))
	}
	return names
}

func sortByName(apps []sdlc.AppVersion) {
	sort.Slice(apps, func(i, j int) bool {
		switch strings.Compare(apps[i].Name, apps[i].Name) {
		case -1:
			return true
		case 1:
			return false
		}
		return apps[i].Name > apps[j].Name
	})
}

func (opt *OptionsTopology) GetComparedTopology() ([]sdlcUtils.Environment, error) {
	masterRepoTmpDir, err := ioutil.TempDir("", "")

	cloneDir, err := gitclient.CloneToDir(GitClient(), opt.GitUrl, masterRepoTmpDir)

	log.WithField("git", opt.GitUrl).WithField("folder", masterRepoTmpDir).Debug("cloning base")
	log.Debug("checking topology...")

	currentHelmState := opt.GetEnvironmentsFromHelmFile(opt.Helmfile, opt.HelmfileDir)
	masterHelmState := opt.GetEnvironmentsFromHelmFile(opt.Helmfile, cloneDir)

	comparedEnvironments := compare(currentHelmState, masterHelmState)
	return comparedEnvironments, err
}

func compare(currentState []sdlcUtils.Environment, newState []sdlcUtils.Environment) []sdlcUtils.Environment {
	var results []sdlcUtils.Environment
	for _, currentEnvState := range currentState {
		var appResults []sdlc.AppVersion
		for _, newEnvState := range newState {
			if currentEnvState.Spec.Namespace == newEnvState.Spec.Namespace {
				if len(currentEnvState.Topology) != len(newEnvState.Topology) {
					currentEnvState.Changed = true
				}
				for _, appVersion := range currentEnvState.Topology {
					if !sdlcUtils.ContainsVersion(appVersion, newEnvState.Topology) {
						appVersion.State = sdlc.StateUpdated
						currentEnvState.Changed = true
					}
					appResults = append(appResults, appVersion)
				}
				currentEnvState.PreviousTopology = newEnvState.Topology
			}
		}
		currentEnvState.Topology = appResults
		results = append(results, currentEnvState)
	}
	return results
}

func (opt *OptionsTopology) GetEnvironmentsFromHelmFile(helmFile string, dir string) []sdlcUtils.Environment {
	gatherHelmfiles := HelmFilePath(helmFile, dir)
	environments := opt.GetEnvironments().Items

	var changedEnvironments []sdlcUtils.Environment
	for _, helmFile := range gatherHelmfiles {
		var environmentVersions []sdlc.AppVersion
		helmState := readHelmState(helmFile)
		releases := helmState.Releases
		if releases != nil {
			for _, release := range releases {
				environmentVersions = append(
					environmentVersions,
					sdlc.AppVersion{
						Name:    release.Name,
						Version: release.Version,
					},
				)
			}
			namespace := helmState.ReleaseSetSpec.OverrideNamespace
			if namespace == "" {
				namespace = findNamespace(releases)
			}
			environment := findEnvironment(environments, namespace)
			environment.Spec.Namespace = namespace
			changedEnvironments = append(changedEnvironments, sdlcUtils.Environment{
				Topology:    environmentVersions,
				Environment: environment,
			})
		}
	}

	return changedEnvironments
}

func findNamespace(releases []state.ReleaseSpec) string {
	if releases == nil {
		return ""
	}
	return releases[0].Namespace
}

func findEnvironment(envs []jxV1.Environment, namesapce string) jxV1.Environment {
	for _, env := range envs {
		if env.Spec.Namespace == namesapce {
			env.Spec.Namespace = namesapce
			return env
		}
	}
	return jxV1.Environment{Spec: jxV1.EnvironmentSpec{
		Namespace: namesapce,
	}}
}

func (opt *OptionsTopology) GetEnvironments() *jxV1.EnvironmentList {
	opt.KubeClient, opt.JxClient, opt.LtClient = sdlcUtils.NewLazyClients(opt.KubeClient, opt.JxClient, opt.LtClient)

	envs, err := opt.JxClient.JenkinsV1().Environments("jx").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic("can't fetch environment list")
	}
	sort.Slice(envs.Items, func(i, j int) bool {
		return envs.Items[i].Spec.Order < envs.Items[j].Spec.Order
	})

	return envs
}

func readHelmState(helmFilePath string) state.HelmState {
	helmState := state.HelmState{}
	err := yaml2s.LoadFile(helmFilePath, &helmState)
	if err != nil {
		panic("HELMFILE")
	}
	return helmState
}

func HelmFilePath(helmfile string, dir string) []string {
	gatherHelmfiles, err := helmfiles.GatherHelmfiles(helmfile, dir)
	var helmFiles = map[string]string{}
	for _, helmfile := range gatherHelmfiles {
		helmFiles[helmfile.Filepath] = helmfile.RelativePathToRoot
	}
	keys := make([]string, 0, len(helmFiles))
	for k := range helmFiles {
		keys = append(keys, k)
	}
	if err != nil {
		panic("cant extract sub helmfiles")
	}
	return keys
}

func GitClient() gitclient.Interface {
	if gitClient == nil {
		if commandRunner == nil {
			commandRunner = cmdrunner.QuietCommandRunner
		}
		gitClient = cli.NewCLIClient("git", commandRunner)
	}
	return gitClient
}
