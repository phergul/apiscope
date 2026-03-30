// Package tui owns terminal rendering and input handling.
//
// The root package contains the Bubble Tea model, grouped shell/runtime state,
// top-level layout, message routing, pane placement, and thin adapters over
// feature packages. Shared UI building blocks and generic shell presentation
// helpers live in widgets/, while pane or view-part behavior and feature-
// specific composition belong in focused feature packages such as operations/,
// details/, request/, response/, and statusbar/.
package tui
