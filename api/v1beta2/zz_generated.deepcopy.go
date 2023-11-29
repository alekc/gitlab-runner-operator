//go:build !ignore_autogenerated

/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1beta2

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesAffinity) DeepCopyInto(out *KubernetesAffinity) {
	*out = *in
	if in.NodeAffinity != nil {
		in, out := &in.NodeAffinity, &out.NodeAffinity
		*out = new(KubernetesNodeAffinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PodAffinity != nil {
		in, out := &in.PodAffinity, &out.PodAffinity
		*out = new(KubernetesPodAffinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PodAntiAffinity != nil {
		in, out := &in.PodAntiAffinity, &out.PodAntiAffinity
		*out = new(KubernetesPodAntiAffinity)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesAffinity.
func (in *KubernetesAffinity) DeepCopy() *KubernetesAffinity {
	if in == nil {
		return nil
	}
	out := new(KubernetesAffinity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesCSI) DeepCopyInto(out *KubernetesCSI) {
	*out = *in
	if in.VolumeAttributes != nil {
		in, out := &in.VolumeAttributes, &out.VolumeAttributes
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesCSI.
func (in *KubernetesCSI) DeepCopy() *KubernetesCSI {
	if in == nil {
		return nil
	}
	out := new(KubernetesCSI)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesConfig) DeepCopyInto(out *KubernetesConfig) {
	*out = *in
	if in.BearerTokenOverwriteAllowed != nil {
		in, out := &in.BearerTokenOverwriteAllowed, &out.BearerTokenOverwriteAllowed
		*out = new(bool)
		**out = **in
	}
	if in.Privileged != nil {
		in, out := &in.Privileged, &out.Privileged
		*out = new(bool)
		**out = **in
	}
	if in.AllowPrivilegeEscalation != nil {
		in, out := &in.AllowPrivilegeEscalation, &out.AllowPrivilegeEscalation
		*out = new(bool)
		**out = **in
	}
	if in.AllowedImages != nil {
		in, out := &in.AllowedImages, &out.AllowedImages
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AllowedServices != nil {
		in, out := &in.AllowedServices, &out.AllowedServices
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PullPolicy != nil {
		in, out := &in.PullPolicy, &out.PullPolicy
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.NodeTolerations != nil {
		in, out := &in.NodeTolerations, &out.NodeTolerations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(KubernetesAffinity)
		(*in).DeepCopyInto(*out)
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TerminationGracePeriodSeconds != nil {
		in, out := &in.TerminationGracePeriodSeconds, &out.TerminationGracePeriodSeconds
		*out = new(int64)
		**out = **in
	}
	if in.PodTerminationGracePeriodSeconds != nil {
		in, out := &in.PodTerminationGracePeriodSeconds, &out.PodTerminationGracePeriodSeconds
		*out = new(int64)
		**out = **in
	}
	if in.CleanupGracePeriodSeconds != nil {
		in, out := &in.CleanupGracePeriodSeconds, &out.CleanupGracePeriodSeconds
		*out = new(int64)
		**out = **in
	}
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(KubernetesPodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.BuildContainerSecurityContext != nil {
		in, out := &in.BuildContainerSecurityContext, &out.BuildContainerSecurityContext
		*out = new(KubernetesContainerSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.HelperContainerSecurityContext != nil {
		in, out := &in.HelperContainerSecurityContext, &out.HelperContainerSecurityContext
		*out = new(KubernetesContainerSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.ServiceContainerSecurityContext != nil {
		in, out := &in.ServiceContainerSecurityContext, &out.ServiceContainerSecurityContext
		*out = new(KubernetesContainerSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = new(KubernetesVolumes)
		(*in).DeepCopyInto(*out)
	}
	if in.HostAliases != nil {
		in, out := &in.HostAliases, &out.HostAliases
		*out = make([]KubernetesHostAliases, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]Service, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CapAdd != nil {
		in, out := &in.CapAdd, &out.CapAdd
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.CapDrop != nil {
		in, out := &in.CapDrop, &out.CapDrop
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DNSConfig != nil {
		in, out := &in.DNSConfig, &out.DNSConfig
		*out = new(KubernetesDNSConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.ContainerLifecycle != nil {
		in, out := &in.ContainerLifecycle, &out.ContainerLifecycle
		*out = new(KubernetesContainerLifecyle)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesConfig.
func (in *KubernetesConfig) DeepCopy() *KubernetesConfig {
	if in == nil {
		return nil
	}
	out := new(KubernetesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesConfigMap) DeepCopyInto(out *KubernetesConfigMap) {
	*out = *in
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesConfigMap.
func (in *KubernetesConfigMap) DeepCopy() *KubernetesConfigMap {
	if in == nil {
		return nil
	}
	out := new(KubernetesConfigMap)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesContainerCapabilities) DeepCopyInto(out *KubernetesContainerCapabilities) {
	*out = *in
	if in.Add != nil {
		in, out := &in.Add, &out.Add
		*out = make([]v1.Capability, len(*in))
		copy(*out, *in)
	}
	if in.Drop != nil {
		in, out := &in.Drop, &out.Drop
		*out = make([]v1.Capability, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesContainerCapabilities.
func (in *KubernetesContainerCapabilities) DeepCopy() *KubernetesContainerCapabilities {
	if in == nil {
		return nil
	}
	out := new(KubernetesContainerCapabilities)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesContainerLifecyle) DeepCopyInto(out *KubernetesContainerLifecyle) {
	*out = *in
	if in.PostStart != nil {
		in, out := &in.PostStart, &out.PostStart
		*out = new(KubernetesLifecycleHandler)
		(*in).DeepCopyInto(*out)
	}
	if in.PreStop != nil {
		in, out := &in.PreStop, &out.PreStop
		*out = new(KubernetesLifecycleHandler)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesContainerLifecyle.
func (in *KubernetesContainerLifecyle) DeepCopy() *KubernetesContainerLifecyle {
	if in == nil {
		return nil
	}
	out := new(KubernetesContainerLifecyle)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesContainerSecurityContext) DeepCopyInto(out *KubernetesContainerSecurityContext) {
	*out = *in
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = new(KubernetesContainerCapabilities)
		(*in).DeepCopyInto(*out)
	}
	if in.Privileged != nil {
		in, out := &in.Privileged, &out.Privileged
		*out = new(bool)
		**out = **in
	}
	if in.RunAsUser != nil {
		in, out := &in.RunAsUser, &out.RunAsUser
		*out = new(int64)
		**out = **in
	}
	if in.RunAsGroup != nil {
		in, out := &in.RunAsGroup, &out.RunAsGroup
		*out = new(int64)
		**out = **in
	}
	if in.RunAsNonRoot != nil {
		in, out := &in.RunAsNonRoot, &out.RunAsNonRoot
		*out = new(bool)
		**out = **in
	}
	if in.ReadOnlyRootFilesystem != nil {
		in, out := &in.ReadOnlyRootFilesystem, &out.ReadOnlyRootFilesystem
		*out = new(bool)
		**out = **in
	}
	if in.AllowPrivilegeEscalation != nil {
		in, out := &in.AllowPrivilegeEscalation, &out.AllowPrivilegeEscalation
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesContainerSecurityContext.
func (in *KubernetesContainerSecurityContext) DeepCopy() *KubernetesContainerSecurityContext {
	if in == nil {
		return nil
	}
	out := new(KubernetesContainerSecurityContext)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesDNSConfig) DeepCopyInto(out *KubernetesDNSConfig) {
	*out = *in
	if in.Nameservers != nil {
		in, out := &in.Nameservers, &out.Nameservers
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]KubernetesDNSConfigOption, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Searches != nil {
		in, out := &in.Searches, &out.Searches
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesDNSConfig.
func (in *KubernetesDNSConfig) DeepCopy() *KubernetesDNSConfig {
	if in == nil {
		return nil
	}
	out := new(KubernetesDNSConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesDNSConfigOption) DeepCopyInto(out *KubernetesDNSConfigOption) {
	*out = *in
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesDNSConfigOption.
func (in *KubernetesDNSConfigOption) DeepCopy() *KubernetesDNSConfigOption {
	if in == nil {
		return nil
	}
	out := new(KubernetesDNSConfigOption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesEmptyDir) DeepCopyInto(out *KubernetesEmptyDir) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesEmptyDir.
func (in *KubernetesEmptyDir) DeepCopy() *KubernetesEmptyDir {
	if in == nil {
		return nil
	}
	out := new(KubernetesEmptyDir)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesHostAliases) DeepCopyInto(out *KubernetesHostAliases) {
	*out = *in
	if in.Hostnames != nil {
		in, out := &in.Hostnames, &out.Hostnames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesHostAliases.
func (in *KubernetesHostAliases) DeepCopy() *KubernetesHostAliases {
	if in == nil {
		return nil
	}
	out := new(KubernetesHostAliases)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesHostPath) DeepCopyInto(out *KubernetesHostPath) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesHostPath.
func (in *KubernetesHostPath) DeepCopy() *KubernetesHostPath {
	if in == nil {
		return nil
	}
	out := new(KubernetesHostPath)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesLifecycleExecAction) DeepCopyInto(out *KubernetesLifecycleExecAction) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesLifecycleExecAction.
func (in *KubernetesLifecycleExecAction) DeepCopy() *KubernetesLifecycleExecAction {
	if in == nil {
		return nil
	}
	out := new(KubernetesLifecycleExecAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesLifecycleHTTPGet) DeepCopyInto(out *KubernetesLifecycleHTTPGet) {
	*out = *in
	if in.HTTPHeaders != nil {
		in, out := &in.HTTPHeaders, &out.HTTPHeaders
		*out = make([]KubernetesLifecycleHTTPGetHeader, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesLifecycleHTTPGet.
func (in *KubernetesLifecycleHTTPGet) DeepCopy() *KubernetesLifecycleHTTPGet {
	if in == nil {
		return nil
	}
	out := new(KubernetesLifecycleHTTPGet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesLifecycleHTTPGetHeader) DeepCopyInto(out *KubernetesLifecycleHTTPGetHeader) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesLifecycleHTTPGetHeader.
func (in *KubernetesLifecycleHTTPGetHeader) DeepCopy() *KubernetesLifecycleHTTPGetHeader {
	if in == nil {
		return nil
	}
	out := new(KubernetesLifecycleHTTPGetHeader)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesLifecycleHandler) DeepCopyInto(out *KubernetesLifecycleHandler) {
	*out = *in
	if in.Exec != nil {
		in, out := &in.Exec, &out.Exec
		*out = new(KubernetesLifecycleExecAction)
		(*in).DeepCopyInto(*out)
	}
	if in.HTTPGet != nil {
		in, out := &in.HTTPGet, &out.HTTPGet
		*out = new(KubernetesLifecycleHTTPGet)
		(*in).DeepCopyInto(*out)
	}
	if in.TCPSocket != nil {
		in, out := &in.TCPSocket, &out.TCPSocket
		*out = new(KubernetesLifecycleTCPSocket)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesLifecycleHandler.
func (in *KubernetesLifecycleHandler) DeepCopy() *KubernetesLifecycleHandler {
	if in == nil {
		return nil
	}
	out := new(KubernetesLifecycleHandler)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesLifecycleTCPSocket) DeepCopyInto(out *KubernetesLifecycleTCPSocket) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesLifecycleTCPSocket.
func (in *KubernetesLifecycleTCPSocket) DeepCopy() *KubernetesLifecycleTCPSocket {
	if in == nil {
		return nil
	}
	out := new(KubernetesLifecycleTCPSocket)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesNodeAffinity) DeepCopyInto(out *KubernetesNodeAffinity) {
	*out = *in
	if in.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		in, out := &in.RequiredDuringSchedulingIgnoredDuringExecution, &out.RequiredDuringSchedulingIgnoredDuringExecution
		*out = new(NodeSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		in, out := &in.PreferredDuringSchedulingIgnoredDuringExecution, &out.PreferredDuringSchedulingIgnoredDuringExecution
		*out = make([]PreferredSchedulingTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesNodeAffinity.
func (in *KubernetesNodeAffinity) DeepCopy() *KubernetesNodeAffinity {
	if in == nil {
		return nil
	}
	out := new(KubernetesNodeAffinity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesPVC) DeepCopyInto(out *KubernetesPVC) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesPVC.
func (in *KubernetesPVC) DeepCopy() *KubernetesPVC {
	if in == nil {
		return nil
	}
	out := new(KubernetesPVC)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesPodAffinity) DeepCopyInto(out *KubernetesPodAffinity) {
	*out = *in
	if in.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		in, out := &in.RequiredDuringSchedulingIgnoredDuringExecution, &out.RequiredDuringSchedulingIgnoredDuringExecution
		*out = make([]PodAffinityTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		in, out := &in.PreferredDuringSchedulingIgnoredDuringExecution, &out.PreferredDuringSchedulingIgnoredDuringExecution
		*out = make([]WeightedPodAffinityTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesPodAffinity.
func (in *KubernetesPodAffinity) DeepCopy() *KubernetesPodAffinity {
	if in == nil {
		return nil
	}
	out := new(KubernetesPodAffinity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesPodAntiAffinity) DeepCopyInto(out *KubernetesPodAntiAffinity) {
	*out = *in
	if in.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		in, out := &in.RequiredDuringSchedulingIgnoredDuringExecution, &out.RequiredDuringSchedulingIgnoredDuringExecution
		*out = make([]PodAffinityTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		in, out := &in.PreferredDuringSchedulingIgnoredDuringExecution, &out.PreferredDuringSchedulingIgnoredDuringExecution
		*out = make([]WeightedPodAffinityTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesPodAntiAffinity.
func (in *KubernetesPodAntiAffinity) DeepCopy() *KubernetesPodAntiAffinity {
	if in == nil {
		return nil
	}
	out := new(KubernetesPodAntiAffinity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesPodSecurityContext) DeepCopyInto(out *KubernetesPodSecurityContext) {
	*out = *in
	if in.FSGroup != nil {
		in, out := &in.FSGroup, &out.FSGroup
		*out = new(int64)
		**out = **in
	}
	if in.RunAsGroup != nil {
		in, out := &in.RunAsGroup, &out.RunAsGroup
		*out = new(int64)
		**out = **in
	}
	if in.RunAsNonRoot != nil {
		in, out := &in.RunAsNonRoot, &out.RunAsNonRoot
		*out = new(bool)
		**out = **in
	}
	if in.RunAsUser != nil {
		in, out := &in.RunAsUser, &out.RunAsUser
		*out = new(int64)
		**out = **in
	}
	if in.SupplementalGroups != nil {
		in, out := &in.SupplementalGroups, &out.SupplementalGroups
		*out = make([]int64, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesPodSecurityContext.
func (in *KubernetesPodSecurityContext) DeepCopy() *KubernetesPodSecurityContext {
	if in == nil {
		return nil
	}
	out := new(KubernetesPodSecurityContext)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesSecret) DeepCopyInto(out *KubernetesSecret) {
	*out = *in
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesSecret.
func (in *KubernetesSecret) DeepCopy() *KubernetesSecret {
	if in == nil {
		return nil
	}
	out := new(KubernetesSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesVolumes) DeepCopyInto(out *KubernetesVolumes) {
	*out = *in
	if in.HostPaths != nil {
		in, out := &in.HostPaths, &out.HostPaths
		*out = make([]KubernetesHostPath, len(*in))
		copy(*out, *in)
	}
	if in.PVCs != nil {
		in, out := &in.PVCs, &out.PVCs
		*out = make([]KubernetesPVC, len(*in))
		copy(*out, *in)
	}
	if in.ConfigMaps != nil {
		in, out := &in.ConfigMaps, &out.ConfigMaps
		*out = make([]KubernetesConfigMap, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make([]KubernetesSecret, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EmptyDirs != nil {
		in, out := &in.EmptyDirs, &out.EmptyDirs
		*out = make([]KubernetesEmptyDir, len(*in))
		copy(*out, *in)
	}
	if in.CSIs != nil {
		in, out := &in.CSIs, &out.CSIs
		*out = make([]KubernetesCSI, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesVolumes.
func (in *KubernetesVolumes) DeepCopy() *KubernetesVolumes {
	if in == nil {
		return nil
	}
	out := new(KubernetesVolumes)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LabelSelector) DeepCopyInto(out *LabelSelector) {
	*out = *in
	if in.MatchLabels != nil {
		in, out := &in.MatchLabels, &out.MatchLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.MatchExpressions != nil {
		in, out := &in.MatchExpressions, &out.MatchExpressions
		*out = make([]NodeSelectorRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LabelSelector.
func (in *LabelSelector) DeepCopy() *LabelSelector {
	if in == nil {
		return nil
	}
	out := new(LabelSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRunner) DeepCopyInto(out *MultiRunner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRunner.
func (in *MultiRunner) DeepCopy() *MultiRunner {
	if in == nil {
		return nil
	}
	out := new(MultiRunner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MultiRunner) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRunnerList) DeepCopyInto(out *MultiRunnerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MultiRunner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRunnerList.
func (in *MultiRunnerList) DeepCopy() *MultiRunnerList {
	if in == nil {
		return nil
	}
	out := new(MultiRunnerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MultiRunnerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRunnerSpec) DeepCopyInto(out *MultiRunnerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRunnerSpec.
func (in *MultiRunnerSpec) DeepCopy() *MultiRunnerSpec {
	if in == nil {
		return nil
	}
	out := new(MultiRunnerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRunnerStatus) DeepCopyInto(out *MultiRunnerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRunnerStatus.
func (in *MultiRunnerStatus) DeepCopy() *MultiRunnerStatus {
	if in == nil {
		return nil
	}
	out := new(MultiRunnerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeSelector) DeepCopyInto(out *NodeSelector) {
	*out = *in
	if in.NodeSelectorTerms != nil {
		in, out := &in.NodeSelectorTerms, &out.NodeSelectorTerms
		*out = make([]NodeSelectorTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeSelector.
func (in *NodeSelector) DeepCopy() *NodeSelector {
	if in == nil {
		return nil
	}
	out := new(NodeSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeSelectorRequirement) DeepCopyInto(out *NodeSelectorRequirement) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeSelectorRequirement.
func (in *NodeSelectorRequirement) DeepCopy() *NodeSelectorRequirement {
	if in == nil {
		return nil
	}
	out := new(NodeSelectorRequirement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeSelectorTerm) DeepCopyInto(out *NodeSelectorTerm) {
	*out = *in
	if in.MatchExpressions != nil {
		in, out := &in.MatchExpressions, &out.MatchExpressions
		*out = make([]NodeSelectorRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.MatchFields != nil {
		in, out := &in.MatchFields, &out.MatchFields
		*out = make([]NodeSelectorRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeSelectorTerm.
func (in *NodeSelectorTerm) DeepCopy() *NodeSelectorTerm {
	if in == nil {
		return nil
	}
	out := new(NodeSelectorTerm)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodAffinityTerm) DeepCopyInto(out *PodAffinityTerm) {
	*out = *in
	if in.LabelSelector != nil {
		in, out := &in.LabelSelector, &out.LabelSelector
		*out = new(LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.NamespaceSelector != nil {
		in, out := &in.NamespaceSelector, &out.NamespaceSelector
		*out = new(LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodAffinityTerm.
func (in *PodAffinityTerm) DeepCopy() *PodAffinityTerm {
	if in == nil {
		return nil
	}
	out := new(PodAffinityTerm)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreferredSchedulingTerm) DeepCopyInto(out *PreferredSchedulingTerm) {
	*out = *in
	in.Preference.DeepCopyInto(&out.Preference)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreferredSchedulingTerm.
func (in *PreferredSchedulingTerm) DeepCopy() *PreferredSchedulingTerm {
	if in == nil {
		return nil
	}
	out := new(PreferredSchedulingTerm)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegisterNewRunnerInfoOptions) DeepCopyInto(out *RegisterNewRunnerInfoOptions) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.Version != nil {
		in, out := &in.Version, &out.Version
		*out = new(string)
		**out = **in
	}
	if in.Revision != nil {
		in, out := &in.Revision, &out.Revision
		*out = new(string)
		**out = **in
	}
	if in.Platform != nil {
		in, out := &in.Platform, &out.Platform
		*out = new(string)
		**out = **in
	}
	if in.Architecture != nil {
		in, out := &in.Architecture, &out.Architecture
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegisterNewRunnerInfoOptions.
func (in *RegisterNewRunnerInfoOptions) DeepCopy() *RegisterNewRunnerInfoOptions {
	if in == nil {
		return nil
	}
	out := new(RegisterNewRunnerInfoOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Runner) DeepCopyInto(out *Runner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Runner.
func (in *Runner) DeepCopy() *Runner {
	if in == nil {
		return nil
	}
	out := new(Runner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Runner) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerList) DeepCopyInto(out *RunnerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Runner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerList.
func (in *RunnerList) DeepCopy() *RunnerList {
	if in == nil {
		return nil
	}
	out := new(RunnerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RunnerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerSpec) DeepCopyInto(out *RunnerSpec) {
	*out = *in
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(string)
		**out = **in
	}
	in.ExecutorConfig.DeepCopyInto(&out.ExecutorConfig)
	if in.Environment != nil {
		in, out := &in.Environment, &out.Environment
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerSpec.
func (in *RunnerSpec) DeepCopy() *RunnerSpec {
	if in == nil {
		return nil
	}
	out := new(RunnerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerStatus) DeepCopyInto(out *RunnerStatus) {
	*out = *in
	if in.LastRegistrationTags != nil {
		in, out := &in.LastRegistrationTags, &out.LastRegistrationTags
		*out = new([]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerStatus.
func (in *RunnerStatus) DeepCopy() *RunnerStatus {
	if in == nil {
		return nil
	}
	out := new(RunnerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Service) DeepCopyInto(out *Service) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Entrypoint != nil {
		in, out := &in.Entrypoint, &out.Entrypoint
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Service.
func (in *Service) DeepCopy() *Service {
	if in == nil {
		return nil
	}
	out := new(Service)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WeightedPodAffinityTerm) DeepCopyInto(out *WeightedPodAffinityTerm) {
	*out = *in
	in.PodAffinityTerm.DeepCopyInto(&out.PodAffinityTerm)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WeightedPodAffinityTerm.
func (in *WeightedPodAffinityTerm) DeepCopy() *WeightedPodAffinityTerm {
	if in == nil {
		return nil
	}
	out := new(WeightedPodAffinityTerm)
	in.DeepCopyInto(out)
	return out
}