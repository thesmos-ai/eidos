// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package outbox recognises the outbox contract — a local append
// paired with the subscribe callable that delivers the appended
// records downstream with at-least-once semantics.
//
// The recognised directive (on the append side) is:
//
//	//+gen:contract outbox role=append subscribe=Subscribe
package outbox
