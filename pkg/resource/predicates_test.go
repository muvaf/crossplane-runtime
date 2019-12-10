/*
Copyright 2019 The Crossplane Authors.

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

package resource

import (
	"github.com/crossplaneio/crossplane-runtime/pkg/resource/fake"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplaneio/crossplane-runtime/pkg/test"
)

func TestAnyOf(t *testing.T) {
	cases := map[string]struct {
		fns  []PredicateFn
		obj  runtime.Object
		want bool
	}{
		"PredicatePasses": {
			fns: []PredicateFn{
				func(obj runtime.Object) bool { return false },
				func(obj runtime.Object) bool { return true },
			},
			want: true,
		},
		"NoPredicatesPass": {
			fns: []PredicateFn{
				func(obj runtime.Object) bool { return false },
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := AnyOf(tc.fns...)(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AnyOf(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestAllOf(t *testing.T) {
	cases := map[string]struct {
		fns  []PredicateFn
		obj  runtime.Object
		want bool
	}{
		"AllPredicatesPass": {
			fns: []PredicateFn{
				func(obj runtime.Object) bool { return true },
				func(obj runtime.Object) bool { return true },
			},
			want: true,
		},
		"NoPredicatesPass": {
			fns: []PredicateFn{
				func(obj runtime.Object) bool { return false },
				func(obj runtime.Object) bool { return false },
			},
			want: false,
		},
		"SomePredicatesPass": {
			fns: []PredicateFn{
				func(obj runtime.Object) bool { return false },
				func(obj runtime.Object) bool { return true },
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := AllOf(tc.fns...)(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AllOf(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestHasManagedResourceReferenceKind(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		c    client.Client
		kind ManagedKind
		want bool
	}{
		"NotAClassReferencer": {
			c:    &test.MockClient{},
			kind: ManagedKind(MockGVK(&fake.MockManaged{})),
			want: false,
		},
		"HasNoResourceReference": {
			obj:  &fake.MockClaim{},
			kind: ManagedKind(MockGVK(&fake.MockManaged{})),
			want: false,
		},
		"HasCorrectResourceReference": {
			obj: &fake.MockClaim{
				MockManagedResourceReferencer: fake.MockManagedResourceReferencer{
					Ref: &corev1.ObjectReference{
						APIVersion: MockGVK(&fake.MockManaged{}).GroupVersion().String(),
						Kind:       MockGVK(&fake.MockManaged{}).Kind,
					},
				},
			},
			kind: ManagedKind(MockGVK(&fake.MockManaged{})),
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := HasManagedResourceReferenceKind(tc.kind)(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasManagedResourceReferenceKind(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestHasClassReferenceKind(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		c    client.Client
		kind ClassKind
		want bool
	}{
		"NotAClassReferencer": {
			c:    &test.MockClient{},
			kind: ClassKind(MockGVK(&fake.MockClass{})),
			want: false,
		},
		"HasNoClassReference": {
			obj:  &fake.MockClaim{},
			kind: ClassKind(MockGVK(&fake.MockClass{})),
			want: false,
		},
		"HasCorrectClassReference": {
			obj: &fake.MockClaim{
				MockClassReferencer: fake.MockClassReferencer{
					Ref: &corev1.ObjectReference{
						APIVersion: MockGVK(&fake.MockClass{}).GroupVersion().String(),
						Kind:       MockGVK(&fake.MockClass{}).Kind,
					},
				},
			},
			kind: ClassKind(MockGVK(&fake.MockClass{})),
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := HasClassReferenceKind(tc.kind)(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasClassReferenceKind(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestIsManagedKind(t *testing.T) {
	cases := map[string]struct {
		kind ManagedKind
		ot   runtime.ObjectTyper
		obj  runtime.Object
		want bool
	}{
		"IsKind": {
			kind: ManagedKind(MockGVK(&fake.MockManaged{})),
			ot:   MockTyper{GVKs: []schema.GroupVersionKind{MockGVK(&fake.MockManaged{})}},
			want: true,
		},
		"IsNotKind": {
			kind: ManagedKind(MockGVK(&fake.MockManaged{})),
			ot:   MockTyper{GVKs: []schema.GroupVersionKind{MockGVK(&fake.MockClaim{})}},
			want: false,
		},
		"ErrorDeterminingKind": {
			kind: ManagedKind(MockGVK(&fake.MockManaged{})),
			ot:   MockTyper{Error: errors.New("boom")},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsManagedKind(tc.kind, tc.ot)(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("IsManagedKind(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestIsControlledByKind(t *testing.T) {
	controller := true

	cases := map[string]struct {
		kind schema.GroupVersionKind
		obj  runtime.Object
		want bool
	}{
		"NoObjectMeta": {
			want: false,
		},
		"NoControllerRef": {
			obj:  &corev1.Secret{},
			want: false,
		},
		"WrongAPIVersion": {
			kind: MockGVK(&fake.MockManaged{}),
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{OwnerReferences: []v1.OwnerReference{
				{
					Kind:       MockGVK(&fake.MockManaged{}).Kind,
					Controller: &controller,
				},
			}}},
			want: false,
		},
		"WrongKind": {
			kind: MockGVK(&fake.MockManaged{}),
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: MockGVK(&fake.MockManaged{}).GroupVersion().String(),
					Controller: &controller,
				},
			}}},
			want: false,
		},
		"IsControlledByKind": {
			kind: MockGVK(&fake.MockManaged{}),
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: MockGVK(&fake.MockManaged{}).GroupVersion().String(),
					Kind:       MockGVK(&fake.MockManaged{}).Kind,
					Controller: &controller,
				},
			}}},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsControlledByKind(tc.kind)(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("IsControlledByKind(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestIsPropagator(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		want bool
	}{
		"NotAnAnnotator": {
			want: false,
		},
		"Missing" + AnnotationKeyPropagateToNamespace: {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateToName: name,
				AnnotationKeyPropagateToUID:  string(uid),
			}}},
			want: false,
		},
		"Missing" + AnnotationKeyPropagateToName: {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateToNamespace: namespace,
				AnnotationKeyPropagateToUID:       string(uid),
			}}},
			want: false,
		},
		"Missing" + AnnotationKeyPropagateToUID: {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateToNamespace: namespace,
				AnnotationKeyPropagateToName:      name,
			}}},
			want: false,
		},
		"IsPropagator": {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateToNamespace: namespace,
				AnnotationKeyPropagateToName:      name,
				AnnotationKeyPropagateToUID:       string(uid),
			}}},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsPropagator()(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("IsPropagator(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestIsPropagated(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		want bool
	}{
		"NotAnAnnotator": {
			want: false,
		},
		"Missing" + AnnotationKeyPropagateFromNamespace: {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateFromName: name,
				AnnotationKeyPropagateFromUID:  string(uid),
			}}},
			want: false,
		},
		"Missing" + AnnotationKeyPropagateFromName: {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateFromNamespace: namespace,
				AnnotationKeyPropagateFromUID:       string(uid),
			}}},
			want: false,
		},
		"Missing" + AnnotationKeyPropagateFromUID: {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateFromNamespace: namespace,
				AnnotationKeyPropagateFromName:      name,
			}}},
			want: false,
		},
		"IsPropagated": {
			obj: &corev1.Secret{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
				AnnotationKeyPropagateFromNamespace: namespace,
				AnnotationKeyPropagateFromName:      name,
				AnnotationKeyPropagateFromUID:       string(uid),
			}}},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsPropagated()(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("IsPropagated(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestHasClassSelector(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		want bool
	}{
		"NotAClassSelector": {
			want: false,
		},
		"NoClassSelector": {
			obj:  &fake.MockClaim{},
			want: false,
		},
		"HasClassSelector": {
			obj:  &fake.MockClaim{MockClassSelector: fake.MockClassSelector{Sel: &v1.LabelSelector{}}},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := HasClassSelector()(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasClassSelector(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestHasNoClassSelector(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		want bool
	}{
		"NotAClassSelector": {
			want: false,
		},
		"NoClassSelector": {
			obj:  &fake.MockClaim{},
			want: true,
		},
		"HasClassSelector": {
			obj:  &fake.MockClaim{MockClassSelector: fake.MockClassSelector{Sel: &v1.LabelSelector{}}},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := HasNoClassSelector()(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasNoClassSelector(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestHasNoClassReference(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		want bool
	}{
		"NotAClassReferencer": {
			want: false,
		},
		"NoClassReference": {
			obj:  &fake.MockClaim{},
			want: true,
		},
		"HasClassReference": {
			obj:  &fake.MockClaim{MockClassReferencer: fake.MockClassReferencer{Ref: &corev1.ObjectReference{}}},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := HasNoClassReference()(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasNoClassReference(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestHasNoMangedResourceReference(t *testing.T) {
	cases := map[string]struct {
		obj  runtime.Object
		want bool
	}{
		"NotAManagedResourceReferencer": {
			want: false,
		},
		"NoManagedResourceReference": {
			obj:  &fake.MockClaim{},
			want: true,
		},
		"HasClassReference": {
			obj:  &fake.MockClaim{MockManagedResourceReferencer: fake.MockManagedResourceReferencer{Ref: &corev1.ObjectReference{}}},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := HasNoManagedResourceReference()(tc.obj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasNoManagedResourecReference(...): -want, +got:\n%s", diff)
			}
		})
	}
}
