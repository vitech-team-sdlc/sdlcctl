package utils

import (
	jxV1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	largetestv1beta1 "github.com/vitech-team/sdlcctl/apis/largetest/v1beta1"
)

type Environment struct {
	Topology []largetestv1beta1.AppVersion `json:"topology"`
	Changed  bool                          `json:"changed"`
	Tested   bool                          `json:"tested"`
	jxV1.Environment
}

func ContainsVersion(version largetestv1beta1.AppVersion, topology []largetestv1beta1.AppVersion) bool {
	for _, tpVersion := range topology {
		if version == tpVersion {
			return true
		}
	}
	return false
}
