// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package watcher recognises the watcher contract — a watch
// callable paired with the trigger callable that fires updates
// the watcher observes.
//
// The recognised directive (on the watch side) is:
//
//	//+gen:contract watcher role=watch trigger=Notify
package watcher
