package topology

import (
	"context"
	"fmt"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/create"
	"github.com/jenkins-x/go-scm/scm"
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/spf13/cobra"
	"github.com/vitech-team/sdlcctl/apis/largetest/v1beta1"
	"github.com/vitech-team/sdlcctl/cmd/utils"
	"io/ioutil"
	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type OptionsTopologyReleaseNotes struct {
	EnvironmentVersion string
	Exclude            []string
	*OptionsTopology
}

func makeReleaseNotesCmd(options *OptionsTopology) *cobra.Command {
	optionReleaseNotes := &OptionsTopologyReleaseNotes{OptionsTopology: options}

	releaseCmd := &cobra.Command{
		Use:     "release-notes",
		Example: "create release notes of current topology ",
		Run: func(cmd *cobra.Command, args []string) {
			err := optionReleaseNotes.ReleaseNotes()
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		},
	}

	releaseCmd.Flags().StringVarP(
		&optionReleaseNotes.Environment, "environment", "", "", "environment to be released",
	)
	releaseCmd.MarkFlagRequired("environment")

	releaseCmd.Flags().StringVarP(
		&optionReleaseNotes.EnvironmentVersion, "version", "", "", "environment version to be released",
	)
	releaseCmd.MarkFlagRequired("version")

	releaseCmd.Flags().StringSliceVarP(
		&optionReleaseNotes.Exclude, "exclude", "", []string{"jx-verify"}, "comma-separated list of applications to exclude (e.g. jenkins-x utility apps)",
	)

	return releaseCmd
}

func (opt *OptionsTopologyReleaseNotes) ReleaseNotes() error {
	envs := opt.GetEnvironments()

	env := findEnvironmentByName(envs.Items, opt.Environment)

	log.WithField("environment", env.ObjectMeta.Name).Info("preparing release notes for:")

	previous, err := opt.LoadPreviousAppReleases(&env)
	if err != nil {
		return err
	}

	current, err := opt.LoadAppReleases(&env)
	if err != nil {
		return err
	}

	releases := opt.CombineAppReleases(current, previous)

	printOutAppReleases(releases)

	topologyReleaseNotes := ""
	for _, release := range releases {
		if release.State != v1beta1.StateSame {
			releaseNotes, err := opt.AppChangeLog(release)
			if err != nil {
				return err
			}

			topologyReleaseNotes += releaseNotes
		}
	}

	if len(strings.Trim(topologyReleaseNotes, " \t\n")) > 0 {
		err = opt.Publish(topologyReleaseNotes)
	} else {
		log.Info("Nothing to publish, environment release notes is empty")
	}

	return err
}

func (opt *OptionsTopologyReleaseNotes) Publish(topologyReleaseNotes string) error {
	scmHelper := scmhelpers.Options{
		Dir:       opt.HelmfileDir,
		SourceURL: opt.GitUrl,
	}

	err := scmHelper.Validate()
	if err != nil {
		return err
	}

	releaseInfo := &scm.ReleaseInput{
		Title:       "Topology release on " + opt.Environment + " - " + opt.EnvironmentVersion,
		Tag:         opt.EnvironmentVersion,
		Description: topologyReleaseNotes,
		Draft:       false,
		Prerelease:  false,
	}

	repoName := scm.Join(scmHelper.Owner, scmHelper.Repository)
	rel, _, err := scmHelper.ScmClient.Releases.Create(context.Background(), repoName, releaseInfo)
	if err != nil {
		return err
	}

	log.WithField("releaseNotesUrl", rel.Link).Info("Release notes has been created")

	return nil
}

func (opt *OptionsTopologyReleaseNotes) LoadPreviousAppReleases(env *jxV1.Environment) (map[utils.AppRelease]utils.Version, error) {
	releases, err := opt.JxClient.JenkinsV1().Releases(env.Spec.Namespace).List(context.TODO(), k8sV1.ListOptions{})

	var appReleases = map[utils.AppRelease]utils.Version{}
	if err == nil {
		for _, release := range releases.Items {
			// TODO
			//release.Spec.Version = "v0.0.1"
			appRelease, appVersion := releaseToAppInfo(release)
			appReleases[appRelease] = appVersion
		}
	}

	return appReleases, err
}

func (opt *OptionsTopologyReleaseNotes) LoadAppReleases(env *jxV1.Environment) (map[utils.AppRelease]utils.Version, error) {
	envConfigRootDir := filepath.Join(opt.HelmfileDir, "config-root", "namespaces", env.Spec.Namespace)

	var appReleases = map[utils.AppRelease]utils.Version{}
	err := filepath.Walk(envConfigRootDir, func(path string, f os.FileInfo, err error) error {
		if err == nil && !f.IsDir() && strings.HasSuffix(f.Name(), "-release.yaml") {
			release, err := tryLoadReleaseResource(path)
			if err == nil {
				appRelease, appVersion := releaseToAppInfo(release)
				appReleases[appRelease] = appVersion
			}
		}
		return err
	})

	return appReleases, err
}

func (opt *OptionsTopologyReleaseNotes) CombineAppReleases(
	current map[utils.AppRelease]utils.Version,
	previous map[utils.AppRelease]utils.Version,
) []utils.AppRelease {
	appReleases := map[string]*utils.AppRelease{}

	excludes := map[string]bool{}
	for _, app := range opt.Exclude {
		excludes[app] = true
	}

	for appRelease, appVersion := range current {
		if _, exists := excludes[appRelease.Name]; !exists {
			appReleases[appRelease.Name] = &utils.AppRelease{
				Name:    appRelease.Name,
				GitUrl:  appRelease.GitUrl,
				Version: appVersion,
			}
		}
	}

	for appRelease, appVersion := range previous {
		if release, ok := appReleases[appRelease.Name]; ok {
			release.PreviousVersion = appVersion
		} else {
			if _, exists := excludes[appRelease.Name]; !exists {
				appReleases[appRelease.Name] = &utils.AppRelease{
					Name:            appRelease.Name,
					GitUrl:          appRelease.GitUrl,
					PreviousVersion: appVersion,
				}
			}
		}
	}

	var releases []utils.AppRelease

	for _, release := range appReleases {
		if len(release.Version.Version) > 0 && len(release.PreviousVersion.Version) > 0 {
			// assume that version is always incremented
			if release.Version != release.PreviousVersion {
				release.State = v1beta1.StateUpdated
			} else {
				release.State = v1beta1.StateSame
			}
		} else if len(release.Version.Version) > 0 {
			release.State = v1beta1.StateAdded
		} else if len(release.PreviousVersion.Version) > 0 {
			release.State = v1beta1.StateRemoved
		} else {
			// how it's possible?
		}

		releases = append(releases, *release)
	}

	sort.Sort(AppReleasesByName{releases})

	return releases
}

func (opt *OptionsTopologyReleaseNotes) AppChangeLog(appRelease utils.AppRelease) (string, error) {
	var releaseNotes []byte

	if appRelease.State == v1beta1.StateRemoved {
		releaseNotes = []byte(fmt.Sprintf(
			"# Component: [%s](%s) - discontinued (previous %s)\n\n",
			appRelease.Name, appRelease.GitUrl, appRelease.PreviousVersion.Version,
		))
	} else {
		client := GitClient()

		gitDir, err := ioutil.TempDir("", "")
		if err != nil {
			panic(err.Error())
		}

		appRelease.Version.Revision = determineTagRevision(client, gitDir, appRelease.GitUrl, appRelease.Version.Version)
		appRelease.PreviousVersion.Revision = determineTagRevision(client, gitDir, appRelease.GitUrl, appRelease.PreviousVersion.Version)

		releaseMd := path.Join(gitDir, rand.String(10)+".md")

		changeLogCmd := create.Options{
			BaseOptions: options.BaseOptions{},
			ScmFactory: scmhelpers.Options{
				Dir:       gitDir,
				SourceURL: appRelease.GitUrl,
			},
			PreviousRevision:   appRelease.PreviousVersion.Revision,
			CurrentRevision:    appRelease.Version.Revision,
			Version:            appRelease.Version.Version,
			OutputMarkdownFile: releaseMd,
			State:              create.State{},
			Header: fmt.Sprintf(
				"# Component: [%s](%s) - %s (previous %s)\n\n",
				appRelease.Name, appRelease.GitUrl, appRelease.Version.Version, appRelease.PreviousVersion.Version,
			),
		}
		err = changeLogCmd.Run()
		if err != nil {
			panic(err.Error())
		}

		releaseNotes, err = ioutil.ReadFile(releaseMd)
		if err != nil {
			panic(err.Error())
		}

		err = os.RemoveAll(gitDir)
		if err != nil {
			panic(err.Error())
		}
	}

	return string(releaseNotes), nil
}

// AppReleases sorting hacks
type AppReleases []utils.AppRelease

func (s AppReleases) Len() int      { return len(s) }
func (s AppReleases) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type AppReleasesByName struct {
	AppReleases
}

func (s AppReleasesByName) Less(i, j int) bool { return s.AppReleases[i].Name < s.AppReleases[j].Name }
