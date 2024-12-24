/*
 * Copyright (c) 2024, HAMi.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plugin

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/Project-HAMi/HAMi/pkg/device/nvidia"
	"github.com/Project-HAMi/HAMi/pkg/util"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func GetNextDeviceRequest(dtype string, p corev1.Pod) (corev1.Container, util.ContainerDevices, error) {
	pdevices, err := util.DecodePodDevices(util.InRequestDevices, p.Annotations)
	if err != nil {
		return corev1.Container{}, util.ContainerDevices{}, err
	}
	klog.Infof("pod annotation decode vaule is %+v", pdevices)
	res := util.ContainerDevices{}

	pd, ok := pdevices[dtype]
	if !ok {
		return corev1.Container{}, res, errors.New("device request not found")
	}
	for ctridx, ctrDevice := range pd {
		if len(ctrDevice) > 0 {
			return p.Spec.Containers[ctridx], ctrDevice, nil
		}
	}
	return corev1.Container{}, res, errors.New("device request not found")
}

func EraseNextDeviceTypeFromAnnotation(dtype string, p corev1.Pod) error {
	pdevices, err := util.DecodePodDevices(util.InRequestDevices, p.Annotations)
	if err != nil {
		return err
	}
	res := util.PodSingleDevice{}
	pd, ok := pdevices[dtype]
	if !ok {
		return errors.New("erase device annotation not found")
	}
	found := false
	for _, val := range pd {
		if found {
			res = append(res, val)
		} else {
			if len(val) > 0 {
				found = true
				res = append(res, util.ContainerDevices{})
			} else {
				res = append(res, val)
			}
		}
	}
	klog.Infoln("After erase res=", res)
	newannos := make(map[string]string)
	newannos[util.InRequestDevices[dtype]] = util.EncodePodSingleDevice(res)
	return util.PatchPodAnnotations(&p, newannos)
}

func GetIndexAndTypeFromUUID(uuid string) (string, int) {
	nvml.Init()
	originuuid := strings.Split(uuid, "[")[0]
	ndev, ret := nvml.DeviceGetHandleByUUID(originuuid)
	if ret != nvml.SUCCESS {
		klog.Error("nvml get handlebyuuid error ret=", ret)
		panic(0)
	}
	Model, ret := ndev.GetName()
	if ret != nvml.SUCCESS {
		klog.Error("nvml get name error ret=", ret)
		panic(0)
	}
	index, ret := ndev.GetIndex()
	if ret != nvml.SUCCESS {
		klog.Error("nvml get index error ret=", ret)
		panic(0)
	}
	return Model, index
}

func GetMigUUIDFromSmiOutput(output string, uuid string, idx int) string {
	migmode := false
	for _, val := range strings.Split(output, "\n") {
		if !strings.Contains(val, "MIG") && strings.Contains(val, uuid) {
			migmode = true
			continue
		}
		if !strings.Contains(val, "MIG") && !strings.Contains(val, uuid) {
			migmode = false
			continue
		}
		if !migmode {
			continue
		}
		klog.Infoln("inspecting", val)
		num := strings.Split(val, "Device")[1]
		num = strings.Split(num, ":")[0]
		num = strings.TrimSpace(num)
		index, err := strconv.Atoi(num)
		if err != nil {
			klog.Fatal("atoi failed num=", num)
		}
		if index == idx {
			outputStr := strings.Split(val, ":")[2]
			outputStr = strings.TrimSpace(outputStr)
			outputStr = strings.TrimRight(outputStr, ")")
			return outputStr
		}
	}
	return ""
}

func GetMigUUIDFromIndex(uuid string, idx int) string {
	nvml.Init()
	originuuid := strings.Split(uuid, "[")[0]
	ndev, ret := nvml.DeviceGetHandleByUUID(originuuid)
	if ret != nvml.SUCCESS {
		klog.Error(`nvml get device uuid error ret=`, ret)
		panic(0)
	}
	migdev, ret := nvml.DeviceGetMigDeviceHandleByIndex(ndev, idx)
	if ret != nvml.SUCCESS {
		klog.Error("nvml get mig dev error ret=", ret, ",idx=", idx, "using nvidia-smi -L for query")
		cmd := exec.Command("nvidia-smi", "-L")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			klog.Fatalf("nvidia-smi -L failed with %s\n", err)
		}
		outStr := stdout.String()
		uuid := GetMigUUIDFromSmiOutput(outStr, originuuid, idx)
		return uuid
	}
	res, ret := migdev.GetUUID()
	if ret != nvml.SUCCESS {
		klog.Error(`nvml get mig uuid error ret=`, ret)
		panic(0)
	}
	return res
}

func GetDeviceNums() int {
	nvml.Init()
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		klog.Error(`nvml get count error ret=`, ret)
	}
	return count
}

func (nv *NvidiaDevicePlugin) ApplyMigTemplate() {
	data, err := yaml.Marshal(nv.migCurrent)
	if err != nil {
		klog.Error("marshal failed", err.Error())
	}
	os.WriteFile("/tmp/migconfig.yaml", data, os.ModePerm)
	cmd := exec.Command("nvidia-mig-parted", "apply", "-f", "/tmp/migconfig.yaml")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		klog.Fatalf("nvidia-mig-parted failed with %s\n", err)
	}
	outStr := stdout.String()
	klog.Infoln("Mig apply", outStr)
}

func (nv *NvidiaDevicePlugin) GenerateMigTemplate(devtype string, devindex int, val util.ContainerDevice) (int, bool) {
	needsreset := false
	position := 0
	for _, migTemplate := range nv.schedulerConfig.MigGeometriesList {
		found := false
		for _, templateDevices := range migTemplate.Models {
			if strings.Contains(devtype, templateDevices) {
				klog.InfoS("type found", "Type", devtype, "Template", templateDevices)
				found = true
				break
			}
		}
		if found {
			templateIdx, pos := util.ExtractMigTemplatesFromUUID(val.UUID)
			position = pos
			v := migTemplate.Geometries[templateIdx]
			for migidx, migpartedDev := range nv.migCurrent.MigConfigs["current"] {
				devicetmpidx := -1
				for didx, device := range migpartedDev.Devices {
					if device == int32(devindex) {
						devicetmpidx = didx
						break
					}
				}
				if devicetmpidx >= 0 {
					migDevices := map[string]int32{}
					for _, migTemplateEntry := range v {
						num, ok := migpartedDev.MigDevices[migTemplateEntry.Name]
						if !ok || num != migTemplateEntry.Count {
							needsreset = true
							migDevices[migTemplateEntry.Name] = migTemplateEntry.Count
						}
					}
					if needsreset {
						if len(migpartedDev.Devices) == 1 {
							nv.migCurrent.MigConfigs["current"][migidx].MigDevices = make(map[string]int32)
							klog.Infoln("replacement for device", migDevices)
							for idx, val := range migDevices {
								nv.migCurrent.MigConfigs["current"][migidx].MigDevices[idx] = val
							}
						} else {
							tmpdevices := []int32{}
							for _, val := range nv.migCurrent.MigConfigs["current"][migidx].Devices {
								if val != int32(devindex) {
									tmpdevices = append(tmpdevices, val)
								}
							}
							nv.migCurrent.MigConfigs["current"][migidx].Devices = make([]int32, 0)
							nv.migCurrent.MigConfigs["current"][migidx].Devices = append(nv.migCurrent.MigConfigs["current"][migidx].Devices, tmpdevices...)
							//migDevices := map[string]int32{}
							nv.migCurrent.MigConfigs["current"] = append(nv.migCurrent.MigConfigs["current"], nvidia.MigConfigSpec{
								Devices:    []int32{int32(devindex)},
								MigEnabled: true,
								MigDevices: migDevices,
							})
						}
					}
					break
				}
			}
		}
	}
	klog.V(3).InfoS("After Generated", nv.migCurrent.MigConfigs["current"])
	return position, needsreset
}

func (nv *NvidiaDevicePlugin) GetContainerDeviceStrArray(c util.ContainerDevices) []string {
	tmp := []string{}
	needsreset := false
	position := 0
	for _, val := range c {
		if !strings.Contains(val.UUID, "[") {
			tmp = append(tmp, val.UUID)
		} else {
			devtype, devindex := GetIndexAndTypeFromUUID(val.UUID)
			position, needsreset = nv.GenerateMigTemplate(devtype, devindex, val)
			if needsreset {
				nv.ApplyMigTemplate()
			}
			tmp = append(tmp, GetMigUUIDFromIndex(val.UUID, position))
		}
	}
	klog.V(3).Infoln("mig current=", nv.migCurrent, ":", needsreset, "position=", position, "uuid lists", tmp)
	return tmp
}