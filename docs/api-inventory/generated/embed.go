// Package generated embeds the Audiobookshelf API inventory produced by the
// scripts under scripts/. Embedding keeps the inventory available to the MCP
// server regardless of the process working directory.
package generated

import _ "embed"

// APIInventoryJSON is the generated Audiobookshelf API inventory, embedded at
// build time from abs-api-inventory.json.
//
//go:embed abs-api-inventory.json
var APIInventoryJSON []byte
