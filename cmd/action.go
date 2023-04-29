// Copyright © 2023 sealos.
//
// Licensed under the Apache License, DefaultVersion 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/cuisongliu/logger"
	"github.com/labring/sealpkg/pkg/apply"
	"github.com/labring/sealpkg/pkg/sync"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var file string
var yamlEnable bool
var applier *apply.Applier

const printInfo = `All Version:
	cri-docker: https://github.com/Mirantis/cri-dockerd/releases
	docker: https://download.docker.com/linux/static/stable/
	containerd: https://github.com/containerd/containerd/releases
	crun: https://github.com/containers/crun/releases
	runc: https://github.com/opencontainers/runc/releases
	sealos: https://github.com/labring/sealos/releases
	crio: https://github.com/cri-o/cri-o/releases
`

// conversionCmd represents the conversion command
var conversionCmd = &cobra.Command{
	Use:     "action",
	Aliases: []string{"a"},
	Short:   "github action output runtime cri and release version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return applier.Apply()
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		logger.Info(printInfo)
		applier = apply.NewApplier()
		if err := applier.WithConfigFiles(file); err != nil {
			return errors.WithMessage(err, "validate config error")
		}
		logger.Debug("begin sync cri all versions")
		s := &sync.Sync{}
		if err := s.Do(); err != nil {
			return errors.WithMessage(err, "sync cri error")
		}
		logger.Debug("sync cri all versions finished")

		if err := applier.WithCRISync(s); err != nil {
			return errors.WithMessage(err, "set sync cri error")
		}

		return applier.WithYaml(yamlEnable)
	},
}

func init() {
	rootCmd.AddCommand(conversionCmd)
	conversionCmd.Flags().StringVarP(&file, "file", "f", "", "config files")
	conversionCmd.Flags().BoolVar(&yamlEnable, "yaml", false, "print yaml")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// conversionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// conversionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
