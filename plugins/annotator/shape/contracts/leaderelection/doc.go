// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package leaderelection recognises the leader-election contract
// — a campaign / resign / isleader triple. The Campaign host
// declares both partners; downstream codegen wires the elected
// participant's leadership lifecycle.
//
// The recognised directive (on the Campaign side) is:
//
//	//+gen:contract leader-election role=campaign resign=Resign isleader=IsLeader
package leaderelection
