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

package apply

import (
	"errors"
	"fmt"
	"github.com/labring-actions/runtime-ctl/pkg/cri"
	"github.com/labring-actions/runtime-ctl/pkg/k8s"
	"github.com/labring-actions/runtime-ctl/pkg/merge"
	v1 "github.com/labring-actions/runtime-ctl/types/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type Applier struct {
	Status      []v1.ComponentAndVersion
	Configs     []v1.RuntimeConfig
	DefaultFile string
	Yaml        bool
	Sync        *cri.Sync
}

func NewApplier() *Applier {
	return &Applier{}
}

func (a *Applier) WithDefaultFile(file string) error {
	if file == "" {
		return errors.New("file not set,please set file retry")
	}
	cfg, err := merge.Merge(file)
	if err != nil {
		return err
	}
	if err = v1.ValidationDefaultComponent(cfg.DefaultVersion); err != nil {
		return err
	}
	const printInfo = `All Default DefaultVersion:
	docker: %s
	containerd: %s
	sealos: %s
	crun: %s
	runc: %s
`
	klog.Infof(printInfo, cfg.DefaultVersion.Docker, cfg.DefaultVersion.Containerd, cfg.DefaultVersion.Sealos, cfg.DefaultVersion.Crun, cfg.DefaultVersion.Runc)
	a.DefaultFile = file
	return nil
}

func (a *Applier) WithCRISync(sync *cri.Sync) error {
	a.Sync = sync
	return nil
}

func (a *Applier) WithYaml(yamlEnable bool) error {
	a.Yaml = yamlEnable
	return nil
}

func (a *Applier) WithConfigFiles(files ...string) error {
	if len(files) <= 0 {
		return errors.New("files not set,please set retry")
	}
	validationFunc := func(index int, r *v1.RuntimeConfig) error {
		if err := v1.ValidationConfigData(r.Config); err != nil {
			return err
		}
		klog.Infof("validate index=%d config data and runtime success", index)
		return nil
	}
	versions := sets.NewString()
	var cfg *v1.RuntimeConfig
	var err error
	for i, f := range files {
		cfg, err = merge.Merge(f, a.DefaultFile)
		if err != nil {
			return err
		}
		if err = validationFunc(i, cfg); err != nil {
			return fmt.Errorf("file is %s is validation error: %+v", f, err)
		}
		if cfg.Config.CRI == nil || len(cfg.Config.CRI) == 0 {
			cfg.Config.CRI = []string{v1.CRIContainerd, v1.CRIDocker, v1.CRICRIO}
		}
	}
	for _, v := range cfg.Config.RuntimeVersion {
		for _, r := range cfg.Config.CRI {
			setKey := fmt.Sprintf("%s-%s-%s", r, cfg.Config.Runtime, v)
			if !versions.Has(setKey) {
				versions.Insert(setKey)
				rt := v1.ComponentAndVersion{
					CRIType:        r,
					Runtime:        cfg.Config.Runtime,
					RuntimeVersion: v,
				}
				rt.CRIRuntime, rt.CRIRuntimeVersion = cri.GetCRIRuntime(r, *cfg.DefaultVersion)
				rt.Sealos = cfg.DefaultVersion.Sealos

				if err = v1.CheckSealosAndRuntime(cfg.Config, cfg.DefaultVersion); err != nil {
					klog.Warningf("check sealos and runtime error: %+v", err)
					continue
				}

				a.Status = append(a.Status, rt)
				a.Configs = append(a.Configs, *cfg)
			}
		}
	}

	return nil
}

func (a *Applier) Apply() error {
	statusList := &v1.RuntimeList{}
	for i, rt := range a.Status {
		localRuntime := a.Status[i]
		switch rt.Runtime {
		case v1.RuntimeK8s:
			kubeBigVersion := v1.ToBigVersion(rt.RuntimeVersion)
			switch rt.CRIType {
			case v1.CRIDocker:
				dockerVersion, criDockerVersion := cri.FetchDockerVersion(rt.RuntimeVersion)
				if dockerVersion != "" {
					localRuntime.CRIVersion = dockerVersion
				} else {
					cfgDocker := a.Configs[i].DefaultVersion.Docker
					localRuntime.CRIVersion = v1.ToBigVersion(cfgDocker)
				}
				versions := a.Sync.Docker[localRuntime.CRIVersion]
				sortList := cri.List(versions)
				newVersion := sortList[len(sortList)-1]
				klog.Infof("docker version is %s, docker using latest version: %s", localRuntime.CRIVersion, newVersion)
				localRuntime.CRIVersion = newVersion
				localRuntime.CRIDockerd = criDockerVersion
			case v1.CRIContainerd:
				localRuntime.CRIVersion = a.Configs[i].DefaultVersion.Containerd
				containerdBigVersion := v1.ToBigVersion(a.Configs[i].DefaultVersion.Containerd)
				if v1.Compare(kubeBigVersion, "1.26") && !v1.Compare(containerdBigVersion, "1.6") {
					return fmt.Errorf("if kubernetes version gt 1.26 your containerd must be gt 1.6")
				}
			case v1.CRICRIO:
				versions := a.Sync.CRIO[kubeBigVersion]
				sortList := cri.List(versions)
				newVersion := sortList[len(sortList)-1]
				klog.Infof("kube version is %s, crio using latest version: %s", rt.RuntimeVersion, newVersion)
				localRuntime.CRIVersion = newVersion
			}

			newVersion, err := k8s.FetchFinalVersion(rt.RuntimeVersion)
			if err != nil {
				return fmt.Errorf("runtime is %s,runtime version is %s,get new version is error: %+v", rt.Runtime, rt.RuntimeVersion, err)
			}
			a.Status[i].RuntimeVersion = newVersion
		default:
			return fmt.Errorf("not found runtime,current version not support")
		}
		if localRuntime.CRIVersion == "" {
			continue
		}
		a.Status[i] = localRuntime
	}
	statusList.Include = a.Status
	if a.Yaml {
		actionYAML, err := yaml.Marshal(statusList)
		if err != nil {
			return err
		}
		fmt.Println(string(actionYAML))
		return nil
	}
	actionJSON, err := json.Marshal(statusList)
	if err != nil {
		return err
	}
	fmt.Println(string(actionJSON))
	return nil
}
