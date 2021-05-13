module github.com/vitech-team/sdlcctl

go 1.15

require (
	github.com/jenkins-x-plugins/jx-changelog v0.0.42
	github.com/jenkins-x-plugins/jx-release-version/v2 v2.4.2
	github.com/jenkins-x/go-scm v1.6.18
	github.com/jenkins-x/jx-api/v4 v4.0.28
	github.com/jenkins-x/jx-gitops v0.2.29
	github.com/jenkins-x/jx-helpers v1.0.88
	github.com/jenkins-x/jx-helpers/v3 v3.0.104
	github.com/olekukonko/tablewriter v0.0.2
	github.com/roboll/helmfile v0.138.4
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	sigs.k8s.io/controller-runtime v0.8.0
)

replace (
	k8s.io/api => k8s.io/api v0.20.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.5
	k8s.io/client-go => k8s.io/client-go v0.20.5
)
