// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package publisher recognises the publisher contract — a publish
// callable paired with the subscribe callable that delivers the
// published events.
//
// The recognised directive (on the publish side) is:
//
//	//+gen:contract publisher role=publish subscribe=Subscribe
package publisher
