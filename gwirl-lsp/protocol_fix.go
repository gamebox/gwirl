package main

import (
	lsp "go.lsp.dev/protocol"
)

type SemanticTokensFullOptions struct {
	// The server supports deltas for full documents.
	Delta bool `json:"delta"`
}

// This type exists as a patch to the protocol packages missing information.
// This is fixed in https://github.com/go-language-server/protocol/pull/49 by
// me.
type SemanticTokensOptions struct {
	lsp.WorkDoneProgressOptions
	// The legend used by the server
	Legend *lsp.SemanticTokensLegend `json:"legend,omitempty"`
	// Server supports providing semantic tokens for a specific range of a document.
	Range bool `json:"range"`
	// Server supports providing semantic tokens for a full document.
	Full interface{} `json:"full,omitempty"`
}
