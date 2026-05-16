// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package saga recognises the saga contract — a step callable
// paired with the compensation callable that reverses its effect
// when a downstream step fails. The validator flags any step
// declaration without a compensation partner.
//
// The recognised directive (on each step side) is:
//
//	//+gen:contract saga role=step compensate=RefundCharge
package saga
