package topology

import (
	"fmt"
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/olekukonko/tablewriter"
	"github.com/vitech-team/sdlcctl/cmd/utils"
	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func releaseToAppInfo(release jxV1.Release) (utils.AppRelease, utils.Version) {
	return utils.AppRelease{
			Name:   release.Spec.Name,
			GitUrl: release.Spec.GitHTTPURL,
		},
		utils.Version{
			Version: release.Spec.Version,
		}
}

func tryLoadReleaseResource(resourceFilePath string) (jxV1.Release, error) {
	typeMeta := k8sV1.TypeMeta{}
	err := yamls.LoadFile(resourceFilePath, &typeMeta)
	if err != nil {
		panic("RESOURCEFILE")
	}

	release := jxV1.Release{}
	if typeMeta.Kind == "Release" {
		err := yamls.LoadFile(resourceFilePath, &release)
		if err != nil {
			panic("RELEASERESOURCE")
		}
	}

	return release, err
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

func determineTagRevision(client gitclient.Interface, gitDir string, gitUrl string, tag string) string {
	err := gitclient.CloneOrPull(client, gitUrl, gitDir)
	if err != nil {
		panic(err.Error())
	}

	revision, err := "", nil
	if tag != "" {
		revision, err = client.Command(gitDir, "rev-list", "-n 1", "tags/"+tag)
	} else {
		revision, err = client.Command(gitDir, "rev-list", "--max-parents=0", "HEAD")
	}
	if err != nil {
		panic(err.Error())
	}

	return revision
}

func printOutAppReleases(releases []utils.AppRelease) {
	var data [][]string

	for _, release := range releases {
		data = append(data, []string{
			release.Name,
			fmt.Sprintf("%v", release.State),
			release.Version.Version,
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
