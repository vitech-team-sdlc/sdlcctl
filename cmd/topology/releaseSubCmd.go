package topology

import (
	"context"
	"fmt"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/create"
	jxAv "github.com/jenkins-x-plugins/jx-release-version/v2/pkg/strategy/auto"
	"github.com/jenkins-x-plugins/jx-release-version/v2/pkg/strategy/fromtag"
	"github.com/jenkins-x-plugins/jx-release-version/v2/pkg/strategy/semantic"
	jxTag "github.com/jenkins-x-plugins/jx-release-version/v2/pkg/tag"
	"github.com/jenkins-x/go-scm/scm"
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/spf13/cobra"
	sdlc "github.com/vitech-team/sdlcctl/apis/topologyrelease/v1beta1"
	"io/ioutil"
	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type State string

const (
	StateAdded   State = "added"
	StateSame    State = "same"
	StateRemoved State = "removed"
	StateUpdated State = "updated"
)

type AppRelease struct {
	Name            string
	GitUrl          string
	NextVersion     sdlc.AppVersion
	PreviousVersion sdlc.AppVersion
	State           State
}

type OptionsTopologyRelease struct {
	Exclude   []string
	Changelog string
	*OptionsTopology
}

func makeReleaseCmd(options *OptionsTopology) *cobra.Command {
	opt := &OptionsTopologyRelease{OptionsTopology: options}

	releaseCmd := &cobra.Command{
		Use:     "release",
		Example: "create release of current topology",
		Run: func(cmd *cobra.Command, args []string) {
			opt.Run()
		},
	}

	releaseCmd.Flags().StringVarP(
		&opt.Environment,
		"environment",
		"",
		"",
		"environment to be released",
	)
	releaseCmd.MarkFlagRequired("environment")

	releaseCmd.Flags().StringVarP(
		&opt.Changelog,
		"changelog",
		"",
		"none",
		"changelog to generate (e.g. none, aggregated, default: none)",
	)

	releaseCmd.Flags().StringSliceVarP(
		&opt.Exclude,
		"exclude",
		"",
		[]string{"jx-verify"},
		"comma-separated list of applications to exclude (e.g. jenkins-x utility apps)",
	)

	return releaseCmd
}

func (opt *OptionsTopologyRelease) Run() {
	envs := opt.GetEnvironments()

	env := findEnvironmentByName(envs.Items, opt.Environment)

	versionTag, prevVersionTag := opt.VersionTags()

	log.WithField("env", env.ObjectMeta.Name).
		WithField("ns", env.Spec.Namespace).
		WithField("version", versionTag).
		WithField("basedOnVersion", prevVersionTag).
		Info("Preparing TopologyRelease")

	topologyRelease := opt.TopologyRelease(&env, versionTag, prevVersionTag)

	switch opt.Changelog {
	case "aggregated":
		log.WithField("environment", env.ObjectMeta.Name).
			Info("Generating Aggregated release notes")
		changelogUrl := opt.AggregatedChangelog(&env, versionTag, prevVersionTag)
		topologyRelease.Spec.ChangelogURL = changelogUrl
		_, err := opt.LtClient.
			TopologyreleaseV1beta1().
			TopologyReleases(env.Spec.Namespace).
			Update(context.TODO(), topologyRelease, k8sV1.UpdateOptions{})
		if err != nil {
			panic(err.Error())
		}
	default:
		log.Info("Release notes won't be generated")
	}
}

func (opt *OptionsTopologyRelease) TopologyRelease(
	env *jxV1.Environment,
	versionTag string,
	prevVersionTag string,
) *sdlc.TopologyRelease {
	appVersions := opt.LoadAppReleases(env)

	topologyRelease := &sdlc.TopologyRelease{
		ObjectMeta: k8sV1.ObjectMeta{
			Namespace: env.Spec.Namespace,
			Name:      makeTopologyName(env, versionTag),
		},
		Spec: sdlc.TopologyReleaseSpec{
			Environment:    env.Name,
			Version:        versionTag,
			BasedOnVersion: prevVersionTag,
			Topology:       appVersions,
		},
	}

	created, err := opt.LtClient.
		TopologyreleaseV1beta1().
		TopologyReleases(env.Spec.Namespace).
		Create(context.TODO(), topologyRelease, k8sV1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	log.WithField("name", created.Name).
		Info("TopologyRelease created has been created")

	return created
}

func (opt *OptionsTopologyRelease) VersionTags() (string, string) {
	strategyOpts := jxAv.Strategy{
		FromTagStrategy: fromtag.Strategy{
			Dir:        opt.HelmfileDir,
			TagPattern: "",
		},
		SemanticStrategy: semantic.Strategy{
			Dir:             opt.HelmfileDir,
			StripPrerelease: false,
		},
	}

	latestVersion, err := strategyOpts.ReadVersion()
	if err != nil {
		panic(err.Error())
	}

	nextVersion, err := strategyOpts.BumpVersion(*latestVersion)
	if err != nil {
		panic(err.Error())
	}

	latestVersionTag := "v" + latestVersion.String()
	nextVersionTag := "v" + nextVersion.String()

	tagOpts := jxTag.Tag{
		FormattedVersion: nextVersionTag,
		Dir:              opt.HelmfileDir,
		PushTag:          true,
	}
	err = tagOpts.TagRemote()
	if err != nil {
		panic(err.Error())
	}

	return nextVersionTag, latestVersionTag
}

func (opt *OptionsTopologyRelease) LoadAppReleases(env *jxV1.Environment) []sdlc.AppVersion {
	envConfigRootDir := filepath.Join(opt.HelmfileDir, "config-root", "namespaces", env.Spec.Namespace)

	excludes := map[string]bool{}
	for _, app := range opt.Exclude {
		excludes[app] = true
	}

	var appVersions []sdlc.AppVersion

	err := filepath.Walk(envConfigRootDir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			panic(err.Error())
		}

		if !f.IsDir() && strings.HasSuffix(f.Name(), "-release.yaml") {
			release := tryLoadReleaseResource(path)

			spec := release.Spec
			if _, exists := excludes[spec.Name]; !exists {
				revision := determineTagRevision(GitClient(), spec.GitHTTPURL, spec.Version)
				appVersions = append(appVersions, sdlc.AppVersion{
					Name:     spec.Name,
					GitURL:   spec.GitHTTPURL,
					Version:  spec.Version,
					Revision: revision,
				})
			}
		}

		return nil
	})
	if err != nil {
		panic(err.Error())
	}

	sortAppVersionsByName(appVersions)

	return appVersions
}

func (opt *OptionsTopologyRelease) AggregatedChangelog(
	env *jxV1.Environment, versionTag string, prevVersionTag string,
) string {
	topologyRelease := opt.GetTopologyReleaseByName(env, versionTag)
	prevTopologyRelease := opt.GetTopologyReleaseByName(env, prevVersionTag)

	appReleases := opt.CombineAppReleases(topologyRelease.Spec.Topology, prevTopologyRelease.Spec.Topology)

	printOutAppReleases(appReleases)

	topologyReleaseNotes := ""
	for _, release := range appReleases {
		if release.State != StateSame {
			changelog, err := opt.AppChangeLog(release)
			if err != nil {
				panic(err.Error())
			}

			topologyReleaseNotes += changelog
		}
	}

	if len(strings.Trim(topologyReleaseNotes, " \t\n")) > 0 {
		return opt.Publish(versionTag, topologyReleaseNotes)
	} else {
		log.Info("Nothing to publish, environment release notes is empty")
	}
	return ""
}

func (opt *OptionsTopologyRelease) GetTopologyReleaseByName(env *jxV1.Environment, prevVersionTag string) *sdlc.TopologyRelease {
	if prevVersionTag == "v0.0.0" {
		return &sdlc.TopologyRelease{}
	} else {
		prevTopologyRelease, err := opt.LtClient.TopologyreleaseV1beta1().
			TopologyReleases(env.Spec.Namespace).
			Get(context.TODO(), makeTopologyName(env, prevVersionTag), k8sV1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}
		return prevTopologyRelease
	}
}

func (opt *OptionsTopologyRelease) CombineAppReleases(
	current []sdlc.AppVersion,
	previous []sdlc.AppVersion,
) []AppRelease {
	appReleases := map[string]*AppRelease{}

	for _, appVersion := range current {
		appReleases[appVersion.Name] = &AppRelease{
			Name:        appVersion.Name,
			GitUrl:      appVersion.GitURL,
			NextVersion: appVersion,
		}
	}

	for _, appVersion := range previous {
		if release, ok := appReleases[appVersion.Name]; ok {
			release.PreviousVersion = appVersion
		} else {
			appReleases[appVersion.Name] = &AppRelease{
				Name:            appVersion.Name,
				GitUrl:          appVersion.GitURL,
				PreviousVersion: appVersion,
			}
		}
	}

	var releases []AppRelease

	for _, release := range appReleases {
		if len(release.NextVersion.Version) > 0 && len(release.PreviousVersion.Version) > 0 {
			// assume that version is always incremented
			if release.NextVersion != release.PreviousVersion {
				release.State = StateUpdated
			} else {
				release.State = StateSame
			}
		} else if len(release.NextVersion.Version) > 0 {
			release.State = StateAdded
		} else if len(release.PreviousVersion.Version) > 0 {
			release.State = StateRemoved
		} else {
			// how it's possible?
		}

		releases = append(releases, *release)
	}

	sortAppReleasesByName(releases)

	return releases
}

func (opt *OptionsTopologyRelease) AppChangeLog(appRelease AppRelease) (string, error) {
	var releaseNotes []byte

	if appRelease.State == StateRemoved {
		releaseNotes = []byte(fmt.Sprintf(
			"# Component: [%s](%s) - discontinued (previous %s)\n\n",
			appRelease.Name, appRelease.GitUrl, appRelease.PreviousVersion.Version,
		))
	} else {
		client := GitClient()

		gitDir := prepareGitRepo(client, &appRelease)

		releaseMd := path.Join(gitDir, rand.String(10)+".md")

		if appRelease.State == StateAdded {
			appRelease.PreviousVersion.Version = "v0.0.0"
			appRelease.PreviousVersion.Revision = determineFirstRevision(client, gitDir)
		}

		changeLogCmd := create.Options{
			BaseOptions: options.BaseOptions{},
			ScmFactory: scmhelpers.Options{
				Dir:       gitDir,
				SourceURL: appRelease.GitUrl,
			},
			PreviousRevision:   appRelease.PreviousVersion.Revision,
			CurrentRevision:    appRelease.NextVersion.Revision,
			Version:            appRelease.NextVersion.Version,
			OutputMarkdownFile: releaseMd,
			State:              create.State{},
			Header: fmt.Sprintf(
				"# Component: [%s](%s) - %s (previous %s)\n\n",
				appRelease.Name, appRelease.GitUrl, appRelease.NextVersion.Version, appRelease.PreviousVersion.Version,
			),
		}
		err := changeLogCmd.Run()
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

func (opt *OptionsTopologyRelease) Publish(versionTag string, topologyChangelog string) string {
	scmHelper := scmhelpers.Options{
		Dir:       opt.HelmfileDir,
		SourceURL: opt.GitUrl,
	}

	err := scmHelper.Validate()
	if err != nil {
		panic(err.Error())
	}

	releaseInput := &scm.ReleaseInput{
		Title:       "Topology release on " + opt.Environment + " - " + versionTag,
		Tag:         versionTag,
		Description: topologyChangelog,
		Draft:       false,
		Prerelease:  false,
	}

	repoName := scm.Join(scmHelper.Owner, scmHelper.Repository)
	rel, _, err := scmHelper.ScmClient.Releases.Create(context.Background(), repoName, releaseInput)
	if err != nil {
		panic(err.Error())
	}

	log.WithField("changelogUrl", rel.Link).Info("Release notes has been created")

	return rel.Link
}
