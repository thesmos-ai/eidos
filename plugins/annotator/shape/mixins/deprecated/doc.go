// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package deprecated recognises the deprecated mixin — the
// assertion that the annotated callable is scheduled for
// removal; downstream test generation may skip the callable or
// emit warnings.
//
// The recognised directive is:
//
//	//+gen:mixin deprecated
package deprecated
