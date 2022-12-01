//go:build !ignore_autogenerated

// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&DynamicCredentialProviderConfig{}, func(obj interface{}) {
		SetObjectDefaults_DynamicCredentialProviderConfig(obj.(*DynamicCredentialProviderConfig))
	})
	return nil
}

func SetObjectDefaults_DynamicCredentialProviderConfig(in *DynamicCredentialProviderConfig) {
	if in.Mirror != nil {
		SetDefaults_MirrorConfig(in.Mirror)
	}
}
