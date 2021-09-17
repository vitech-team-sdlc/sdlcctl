package topology

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/create"
	"github.com/jenkins-x-plugins/jx-release-version/v2/pkg/strategy/auto"
	"github.com/jenkins-x-plugins/jx-release-version/v2/pkg/strategy/semantic"
	jxTag "github.com/jenkins-x-plugins/jx-release-version/v2/pkg/tag"
	"github.com/jenkins-x/go-scm/scm"
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	sdlc "github.com/vitech-team/sdlcctl/apis/topologyrelease/v1beta1"
	"io/ioutil"
	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
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
		[]string{
			"jx-verify",
			"jx-preview",
			"lighthouse",
		},
		"comma-separated list of applications to exclude (e.g. jenkins-x utility apps)",
	)

	return releaseCmd
}

func (opt *OptionsTopologyRelease) Run() {
	envs := opt.GetEnvironments()

	var prevEnv *jxV1.Environment = nil
	for _, env := range envs.Items {
		envLog := log.
			WithField("environment", env.Name).
			WithField("namespace", env.Spec.Namespace)
		if env.Spec.Kind == jxV1.EnvironmentKindTypePermanent {
			version, prevVersion, prevEnvVersion := opt.DetermineChanges(&env, prevEnv)

			envLog = envLog.WithField("version", version)

			envLog.
				WithField("prevVersion", prevVersion).
				WithField("prevEnvVersion", prevEnvVersion).
				Info("Preparing TopologyRelease")

			topologyRelease := opt.TopologyRelease(&env, version, prevVersion, prevEnvVersion, envLog)

			if topologyRelease != nil {
				envLog.Info("Creating release tag")
				opt.TagVersion(version)

				switch opt.Changelog {
				case "aggregated":
					envLog.Info("Generating Aggregated release notes")
					changelogUrl := opt.AggregatedChangelog(&env, version, prevVersion, envLog)
					topologyRelease.Spec.ChangelogURL = changelogUrl
					opt.UpdateTopologyRelease(&env, topologyRelease)
				default:
					envLog.Info("Release notes won't be generated")
				}
			}

			prevEnv = env.DeepCopy()
		} else {
			envLog.Info("Skipping release creation")
		}
	}
}

func (opt *OptionsTopologyRelease) TopologyRelease(env *jxV1.Environment, version *semver.Version, prevVersion *semver.Version, prevEnvVersion *semver.Version, envLog *logrus.Entry) *sdlc.TopologyRelease {
	appVersions := opt.LoadAppReleases(env)

	prevAppVersions := opt.GetTopologyRelease(env, prevVersion).Spec.Topology
	sortAppVersionsByName(prevAppVersions)

	var topologyRelease *sdlc.TopologyRelease = nil
	if !reflect.DeepEqual(prevAppVersions, appVersions) {
		prevVersionTag := ""
		if prevVersion != nil {
			prevVersionTag = prevVersion.Original()
		}
		prevEnvVersionTag := ""
		if prevEnvVersion != nil {
			prevEnvVersionTag = prevEnvVersion.Original()
		}
		topologyRelease = &sdlc.TopologyRelease{
			ObjectMeta: k8sV1.ObjectMeta{
				Name:      version.String(),
				Namespace: env.Spec.Namespace,
			},
			Spec: sdlc.TopologyReleaseSpec{
				Environment:    env.Name,
				Version:        version.Original(),
				PrevVersion:    prevVersionTag,
				PrevEnvVersion: prevEnvVersionTag,
				Topology:       appVersions,
			},
		}
		topologyRelease = opt.CreateTopologyRelease(env, topologyRelease)
		envLog.
			WithField("name", topologyRelease.Name).
			Info("TopologyRelease created has been created")
	} else {
		envLog.Info("TopologyRelease won't be created since none application changed")
	}

	return topologyRelease
}

func (opt *OptionsTopologyRelease) TagVersion(version *semver.Version) {
	tagOpts := jxTag.Tag{
		FormattedVersion: version.Original(),
		Dir:              opt.HelmfileDir,
		PushTag:          true,
	}
	err := tagOpts.TagRemote()
	if err != nil {
		panic(err.Error())
	}
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

func (opt *OptionsTopologyRelease) AggregatedChangelog(env *jxV1.Environment, version *semver.Version, prevVersion *semver.Version, envLog *logrus.Entry) string {
	topologyRelease := opt.GetTopologyRelease(env, version)
	prevTopologyRelease := opt.GetTopologyRelease(env, prevVersion)

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
		return opt.Publish(version, topologyReleaseNotes, envLog)
	} else {
		envLog.Info("Nothing to publish, environment release notes is empty")
	}
	return ""
}

func (opt *OptionsTopologyRelease) GetTopologyRelease(
	env *jxV1.Environment,
	version *semver.Version,
) *sdlc.TopologyRelease {
	if version == nil {
		return &sdlc.TopologyRelease{}
	} else {
		topologyRelease, err := opt.LtClient.TopologyreleaseV1beta1().
			TopologyReleases(env.Spec.Namespace).
			Get(context.TODO(), version.String(), k8sV1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}
		return topologyRelease
	}
}

func (opt *OptionsTopologyRelease) ListTopologyRelease(
	env *jxV1.Environment,
) []sdlc.TopologyRelease {
	topologyReleaseList, err := opt.LtClient.TopologyreleaseV1beta1().
		TopologyReleases(env.Spec.Namespace).
		List(context.TODO(), k8sV1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return topologyReleaseList.Items
}

func (opt *OptionsTopologyRelease) CreateTopologyRelease(
	env *jxV1.Environment,
	topologyRelease *sdlc.TopologyRelease,
) *sdlc.TopologyRelease {
	created, err := opt.LtClient.
		TopologyreleaseV1beta1().
		TopologyReleases(env.Spec.Namespace).
		Create(context.TODO(), topologyRelease, k8sV1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	return created
}

func (opt *OptionsTopologyRelease) UpdateTopologyRelease(
	env *jxV1.Environment,
	topologyRelease *sdlc.TopologyRelease,
) *sdlc.TopologyRelease {
	updated, err := opt.LtClient.
		TopologyreleaseV1beta1().
		TopologyReleases(env.Spec.Namespace).
		Update(context.TODO(), topologyRelease, k8sV1.UpdateOptions{})
	if err != nil {
		panic(err.Error())
	}
	return updated
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

func (opt *OptionsTopologyRelease) Publish(version *semver.Version, topologyChangelog string, envLog *logrus.Entry) string {
	scmHelper := scmhelpers.Options{
		Dir:       opt.HelmfileDir,
		SourceURL: opt.GitUrl,
	}

	err := scmHelper.Validate()
	if err != nil {
		panic(err.Error())
	}

	releaseInput := &scm.ReleaseInput{
		Title:       "Topology release " + version.Original(),
		Tag:         version.Original(),
		Description: topologyChangelog,
		Draft:       false,
		Prerelease:  false,
	}

	repoName := scm.Join(scmHelper.Owner, scmHelper.Repository)
	rel, _, err := scmHelper.ScmClient.Releases.Create(context.Background(), repoName, releaseInput)
	if err != nil {
		panic(err.Error())
	}

	envLog.WithField("changelogUrl", rel.Link).Info("Release notes has been created")

	return rel.Link
}

func (opt *OptionsTopologyRelease) DetermineChanges(
	env *jxV1.Environment,
	prevEnv *jxV1.Environment,
) (*semver.Version, *semver.Version, *semver.Version) {
	var prevVersion *semver.Version = nil

	prevReleases := opt.ListTopologyRelease(env)
	if len(prevReleases) > 0 {
		if len(prevReleases) > 1 {
			sort.SliceStable(prevReleases, func(i, j int) bool {
				v1 := semver.MustParse(prevReleases[i].Spec.Version)
				v2 := semver.MustParse(prevReleases[j].Spec.Version)
				return v1.GreaterThan(v2)
			})
		}
		prevVersion = semver.MustParse(prevReleases[0].Spec.Version)
	}

	var prevEnvVersion *semver.Version = nil
	if prevEnv != nil {
		// look up for the same topology layout on previous environment
		appVersions := opt.LoadAppReleases(env)
		prevEnvReleases := opt.ListTopologyRelease(prevEnv)
		sort.SliceStable(prevEnvReleases, func(i, j int) bool {
			v1 := semver.MustParse(prevEnvReleases[i].Spec.Version)
			v2 := semver.MustParse(prevEnvReleases[j].Spec.Version)
			return v1.GreaterThan(v2)
		})

		for _, prevEnvRelease := range prevEnvReleases {
			prevAppVersions := prevEnvRelease.Spec.Topology
			sortAppVersionsByName(prevAppVersions)
			if reflect.DeepEqual(prevAppVersions, appVersions) {
				prevEnvVersion = semver.MustParse(prevEnvRelease.Spec.Version)
				break
			}
		}
	}

	var nextVersion *semver.Version = nil
	if prevEnv == nil {
		// prevEnv doesn't exist - increment according to conventional commit rules
		if prevVersion == nil {
			nextVersion = semver.MustParse("v0.0.1-" + env.Name)
		} else {
			strategy := auto.Strategy{
				SemanticStrategy: semantic.Strategy{
					Dir:             opt.HelmfileDir,
					StripPrerelease: false,
				},
			}
			tmpVer1, err := prevVersion.SetPrerelease("")
			if err != nil {
				panic(err.Error())
			}
			tmpVer2, err := strategy.BumpVersion(tmpVer1)
			if err != nil {
				panic(err.Error())
			}
			tmpVer3, err := tmpVer2.SetPrerelease(env.Name)
			if err != nil {
				panic(err.Error())
			}
			nextVersion = &tmpVer3
		}
	} else {
		if prevEnvVersion == nil {
			// TODO: probably allow some kind of not-promoted releases
			panic("Unable to determine previous topology release with the same application layout")
		} else {
			// prevEnv exists and prevEnvVersion exists - (promoted topology release) keep the same version but change `pre` suffix only
			tmpNextVersion, err := prevEnvVersion.SetPrerelease(env.Name)
			if err != nil {
				panic(err.Error())
			}
			nextVersion = &tmpNextVersion
		}
	}

	return nextVersion, prevVersion, prevEnvVersion
}
