/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, DefaultVersion 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"fmt"
	"runtime"
)

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() Output {
	// These variables typically come from -ldflags settings and in
	// their absence fallback to the settings in ./base.go
	return Output{
		Version: Info{
			GitVersion: gitVersion,
			GitCommit:  gitCommit,
			BuildDate:  buildDate,
			GoVersion:  runtime.Version(),
			Compiler:   runtime.Compiler,
			Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		},
		DefaultVersion: DefaultVersion{
			CRIDockerd3x: CRIDockerd3x,
			CRIDockerd2x: CRIDockerd2x,
			Dockerd18:    Docker18,
			Dockerd19:    Docker19,
			Dockerd20:    Docker20,
		},
	}
}
