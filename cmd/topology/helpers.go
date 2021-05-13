package topology

import (
	"fmt"
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/olekukonko/tablewriter"
	"github.com/vitech-team/sdlcctl/apis/topologyrelease/v1beta1"
	"io/ioutil"
	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sort"
	"strings"
)

func tryLoadReleaseResource(resourceFilePath string) jxV1.Release {
	typeMeta := k8sV1.TypeMeta{}
	err := yamls.LoadFile(resourceFilePath, &typeMeta)
	if err != nil {
		panic(err.Error())
	}

	release := jxV1.Release{}
	if typeMeta.Kind == "Release" {
		err = yamls.LoadFile(resourceFilePath, &release)
		if err != nil {
			panic(err.Error())
		}
	}

	return release
}

func findEnvironmentByName(envs []jxV1.Environment, environment string) jxV1.Environment {
	idx := -1
	for i, env := range envs {
		if env.ObjectMeta.Name == environment {
			idx = i
			break
		}
	}

	if idx == -1 {
		panic("Unable to find environment: " + environment)
	}

	return envs[idx]
}

func determineTagRevision(client gitclient.Interface, gitUrl string, tag string) string {
	gitDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err.Error())
	}

	_, err = client.Command(gitDir, "clone", "--branch", tag, "--depth=1", "--bare", "--quiet", gitUrl, "./")
	if err != nil {
		panic(err.Error())
	}

	revision, err := client.Command(gitDir, "log", "-1", "--pretty=%H")
	if err != nil {
		panic(err.Error())
	}

	err = os.RemoveAll(gitDir)
	if err != nil {
		panic(err.Error())
	}

	return revision
}

func determineFirstRevision(client gitclient.Interface, gitDir string) string {
	revision, err := client.Command(gitDir, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		panic(err.Error())
	}

	return revision
}

func prepareGitRepo(client gitclient.Interface, appRelease *AppRelease) string {
	gitDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err.Error())
	}

	err = gitclient.CloneOrPull(client, appRelease.GitUrl, gitDir)
	if err != nil {
		panic(err.Error())
	}

	return gitDir
}

func sortAppVersionsByName(apps []v1beta1.AppVersion) {
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

func makeTopologyName(env *jxV1.Environment, versionTag string) string {
	return fmt.Sprintf("%s-%s", env.Name, versionTag)
}

func sortAppReleasesByName(apps []AppRelease) {
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

func printOutAppReleases(releases []AppRelease) {
	var data [][]string

	for _, release := range releases {
		data = append(data, []string{
			release.Name,
			fmt.Sprintf("%v", release.State),
			release.NextVersion.Version,
			release.PreviousVersion.Version,
			release.GitUrl,
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Application", "State", "Version", "Prev Version", "Repository"})

	table.AppendBulk(data)
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.SetCenterSeparator("|")
	table.Render()
}
