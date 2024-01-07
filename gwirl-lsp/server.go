package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/gamebox/gwirl/internal/parser"
	"go.lsp.dev/jsonrpc2"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func readHeaders(r *bufio.Reader) (map[string]string, error) {
	headers := map[string]string{}
	bytes := make([]byte, 0, 100)
	for {
		str, err := r.ReadString('\n')
		if err != nil && str != "" {
			r.Read(bytes)
			return headers, jsonrpc2.NewError(jsonrpc2.ParseError, "Could not parse headers")
		}
		if err != nil && str == "" {
			r.Read(bytes)
			return headers, nil
		}
		if strings.HasPrefix(str, "Content-Length:") && strings.HasSuffix(str, "\r\n") {
			segments := strings.Split(str, ":")
			value := strings.TrimLeft(segments[1], " ")
			value = strings.TrimRight(value, "\r\n")
			headers["Content-Length"] = value
		}
		if str == "\r\n" {
			return headers, nil
		}
	}
}

func (s *GwirlLspServer) readMessage() (string, []byte, error) {
	headers, err := readHeaders(s.reader)
	if err != nil {
		return "", nil, jsonrpc2.NewError(jsonrpc2.ParseError, fmt.Sprintf("Failed to read headers: %v", err))
	}
	contentLength := headers["Content-Length"]
	if contentLength == "0" {
		return "", nil, nil
	}
	length, err := strconv.ParseUint(contentLength, 10, 64)
	if err != nil {
		length = 0
	}
	if length == 0 {
		return "", []byte{}, nil
	}
	var bytesRead uint64 = 0
	bytes := make([]byte, 0, 64000)
	for bytesRead < length {
		byte, err := s.reader.ReadByte()
		if err != nil {
			return "", nil, jsonrpc2.NewError(jsonrpc2.ParseError, "Could not read message body")
		}
		bytes = append(bytes, byte)
		bytesRead += 1
	}

	var jsonMessage Message
	err = json.Unmarshal(bytes, &jsonMessage)
	if err != nil {
		return "", nil, jsonrpc2.NewError(jsonrpc2.ParseError, "Could not decode message body")
	}

	return jsonMessage.Method, bytes, nil
}

type Message struct {
	Version string `json:"version"`
	Method  string `json:"method"`
}

func (s *GwirlLspServer) WriteMessage(msg any) {
	bytes, err := json.Marshal(msg)
	if err != nil {
		s.logf("Failed to write message msg=%v err=%v", msg, err)
		return
	}
	length := len(bytes)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.writer.Write([]byte("Content-Length: "))
	s.writer.Write([]byte(fmt.Sprintf("%d", length)))
	s.writer.Write([]byte("\r\n\r\n"))
	s.writer.Write(bytes)
}

type LspRequest struct {
	Id     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type LspResponse struct {
	Id     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *jsonrpc2.Error `json:"error,omitempty"`
}

func handleRequest[I any, O any](s *GwirlLspServer, method string, handler func(context.Context, *I) (O, error), ctx context.Context, bytes []byte) {
	defer func() {
		if err := recover(); err != nil {
			s.sendNotification(lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
				Message: fmt.Sprintf("Server crashed with error: %v", err),
				Type:    lsp.MessageTypeInfo,
			})
		}
	}()
	var request LspRequest
	err := json.Unmarshal(bytes, &request)
	if err != nil {
		s.logf(`Method "%s" failed to unmarshal request: %e`, method, err)
		e := jsonrpc2.NewError(jsonrpc2.ParseError, fmt.Sprintf("Error handling method \"%s\": %v", method, err))
		s.WriteMessage(e)
		return
	}
	var params I
	err = json.Unmarshal(request.Params, &params)
	if err != nil {
		s.logf(`Method "%s" failed to unmarshal params: %e`, method, err)
		e := jsonrpc2.NewError(jsonrpc2.ParseError, fmt.Sprintf("Error handling method \"%s\": %v", method, err))
		s.WriteMessage(e)
		return
	}
	result, err := handler(ctx, &params)
	if err != nil {
		s.logf(`Method "%s" handler returned error: %e`, method, err)
		s.WriteMessage(jsonrpc2.NewError(jsonrpc2.InternalError, fmt.Sprintf("Error handling method \"%s\": %v", method, err)))
		return
	}
	r, err := json.Marshal(result)
	if err != nil {
		s.logf(`Method "%s" failed to marshal result: %e`, method, err)
		s.WriteMessage(jsonrpc2.NewError(jsonrpc2.InternalError, fmt.Sprintf("Error writing response for method \"%s\": %v", method, err)))
		return
	}
	s.logf(`Method "%s" responding successfully`, method)
	s.WriteMessage(LspResponse{
		Id:     request.Id,
		Result: r,
	})
}

func handleNotification[I any](s *GwirlLspServer, method string, handler func(context.Context, *I) error, ctx context.Context, bytes []byte) {
	var request LspRequest
	err := json.Unmarshal(bytes, &request)
	if err != nil {
		s.logf("Error handling method : %v", err)
		return
	}
	var params I
	err = json.Unmarshal(request.Params, &params)
	if err != nil {
		s.logf("Error handling method : %v", err)
		return
	}
	err = handler(ctx, &params)
	if err != nil {
		s.logf("Error handling method initialize: %v", err)
	}
}

type GwirlLspServer struct {
	writer       io.Writer
	reader       *bufio.Reader
	logWriter    io.Writer
	mutex        *sync.Mutex
	logMutex     *sync.Mutex
	openedFiles  []string
	fileContents map[uri.URI]string
	context      context.Context
	parser       *parser.Parser2
}

func NewGwirlLspServer(writer io.Writer, logWriter io.Writer, reader *bufio.Reader, context context.Context) *GwirlLspServer {
	log.SetOutput(logWriter)
	p := parser.NewParser2("")
	s := GwirlLspServer{
		writer:       writer,
		reader:       reader,
		logWriter:    logWriter,
		mutex:        &sync.Mutex{},
		logMutex:     &sync.Mutex{},
		openedFiles:  make([]string, 10, 100),
		fileContents: make(map[uri.URI]string, 10),
		context:      context,
		parser:       &p,
	}

	s.log("GwirlLspServer: Start")

	return &s
}

type LspNotification struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (s *GwirlLspServer) Handle() {
	for {
		method, bytes, err := s.readMessage()
		if err != nil {
			s.WriteMessage(err)
			return
		}
		if method != "" {
			s.log("==========================================")
			s.logf("Received message for method \"%s\"", method)
			s.log("------------------------------------------")
		}
		switch method {
		case lsp.MethodInitialize:
			go handleRequest[lsp.InitializeParams, *lsp.InitializeResult](s, method, s.Initialize, s.context, bytes)
		case lsp.MethodInitialized:
			go handleNotification[lsp.InitializedParams](s, method, s.Initialized, s.context, bytes)
		case lsp.MethodShutdown:
			s.log("Shutting down...")
			os.Exit(0)
		case lsp.MethodTextDocumentDidOpen:
			go handleNotification[lsp.DidOpenTextDocumentParams](s, method, s.DidOpen, s.context, bytes)
		case lsp.MethodTextDocumentDidChange:
			go handleNotification[lsp.DidChangeTextDocumentParams](s, method, s.DidChange, s.context, bytes)
		case lsp.MethodTextDocumentCompletion:
			go handleRequest[lsp.CompletionParams, *lsp.CompletionList](s, method, s.Completion, s.context, bytes)
		case lsp.MethodTextDocumentDefinition:
			go handleRequest[lsp.DefinitionParams, []lsp.Location](s, method, s.Definition, s.context, bytes)
		case lsp.MethodSemanticTokensFull:
			go handleRequest[lsp.SemanticTokensParams, *lsp.SemanticTokens](s, method, s.SemanticTokensFull, s.context, bytes)
		case lsp.MethodTextDocumentHover:
			go handleRequest[lsp.HoverParams, *lsp.Hover](s, method, s.Hover, s.context, bytes)
		default:
			s.logf("Not handling method \"%s\"", method)
		}
	}
}

func (s *GwirlLspServer) log(message string) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()
	log.Println("[SERVER]: " + message)
}

func (s *GwirlLspServer) logf(format string, data ...any) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()
	log.Printf("[SERVER]: "+format, data...)
}

func (s *GwirlLspServer) sendNotification(method string, notification any) {
	if s.writer != nil {
		params, err := json.Marshal(notification)
		if err != nil {
			s.logf("Could not send message: %v", notification)
		}
		n := LspNotification{
			Version: "2.0",
			Method:  method,
			Params:  params,
		}
		s.logf("Sending notification \"%s\"", n.Method)
		s.WriteMessage(n)
	}
}

func (s *GwirlLspServer) RetrieveWorkspaceTemplates(root string) {
	entries := make([]string, 0, 100)
	entries = append(entries, root)
	for idx := 0; idx < len(entries); idx += 1 {
		entry, err := os.Stat(entries[idx])
		if err != nil {
			continue
		}
		if entry.IsDir() {
			es, err := os.ReadDir(entries[idx])
			if err != nil {
				if strings.HasSuffix(entry.Name(), "/templates") {
					s.logf("Could not load directory %s: %v", entry, err)
				}
				continue
			}
			for _, e := range es {
				entries = append(entries, path.Join(entries[idx], e.Name()))
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), ".gwirl") {
			s.logf("Loading %s", entries[idx])
			contents, err := os.ReadFile(entries[idx])
			if err != nil {
				continue
			}
			s.fileContents[uri.File(entries[idx])] = string(contents)
		}
	}
	s.logf("Loaded %d templates", len(s.fileContents))
}

func (s *GwirlLspServer) Initialize(ctx context.Context, params *lsp.InitializeParams) (result *lsp.InitializeResult, err error) {
	s.logf("Initialized with rootURI \"%s\" and rootPath \"%s\"", params.RootURI.Filename(), params.RootPath)
	s.RetrieveWorkspaceTemplates(params.RootURI.Filename())
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: lsp.TextDocumentSyncKindFull,
			HoverProvider: lsp.HoverOptions{
				WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{},
			},
			DefinitionProvider: lsp.DefinitionOptions{
				WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{},
			},
			CompletionProvider: &lsp.CompletionOptions{
				ResolveProvider:   true,
				TriggerCharacters: []string{"@", "a-zA-Z("},
			},
			SemanticTokensProvider: SemanticTokensOptions{
				WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
				Legend: &lsp.SemanticTokensLegend{
					TokenTypes:     SemanticTokenLegend,
					TokenModifiers: []lsp.SemanticTokenModifiers{},
				},
				Range: false,
				Full:  &SemanticTokensFullOptions{Delta: false},
			},
		},
	}, nil
}
func (s *GwirlLspServer) Initialized(ctx context.Context, params *lsp.InitializedParams) (err error) {
	s.logf("Server initialized: %v", params)
	if len(s.openedFiles) == 0 {
		return
	}
	return
}
func (s *GwirlLspServer) Shutdown(ctx context.Context) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) Exit(ctx context.Context) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) WorkDoneProgressCancel(ctx context.Context, params *lsp.WorkDoneProgressCancelParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) LogTrace(ctx context.Context, params *lsp.LogTraceParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) SetTrace(ctx context.Context, params *lsp.SetTraceParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) CodeAction(ctx context.Context, params *lsp.CodeActionParams) (result []lsp.CodeAction, err error) {
	return nil, nil
}
func (s *GwirlLspServer) CodeLens(ctx context.Context, params *lsp.CodeLensParams) (result []lsp.CodeLens, err error) {
	return nil, nil
}
func (s *GwirlLspServer) CodeLensResolve(ctx context.Context, params *lsp.CodeLens) (result *lsp.CodeLens, err error) {
	return nil, nil
}
func (s *GwirlLspServer) ColorPresentation(ctx context.Context, params *lsp.ColorPresentationParams) (result []lsp.ColorPresentation, err error) {
	return nil, nil
}

var baseCompletionItems = []lsp.CompletionItem{
	{
		Label:            "if",
		InsertText:       "if $1 {$2}",
		InsertTextMode:   lsp.InsertTextModeAsIs,
		InsertTextFormat: lsp.InsertTextFormatSnippet,
		Kind:             lsp.CompletionItemKindKeyword,
	},
	{
		Label:            "for",
		InsertText:       "for $1 {$2}",
		InsertTextMode:   lsp.InsertTextModeAsIs,
		InsertTextFormat: lsp.InsertTextFormatSnippet,
		Kind:             lsp.CompletionItemKindKeyword,
	},
	{
		Label:            "block",
		InsertText:       "{$1}",
		InsertTextMode:   lsp.InsertTextModeAsIs,
		InsertTextFormat: lsp.InsertTextFormatSnippet,
		Kind:             lsp.CompletionItemKindSnippet,
	},
}

func (s *GwirlLspServer) Completion(ctx context.Context, params *lsp.CompletionParams) (result *lsp.CompletionList, err error) {
	list := lsp.CompletionList{}
	presult := s.parser.Parse(s.fileContents[params.TextDocument.URI], params.TextDocument.URI.Filename())
	tParams := GetTemplateParamNames(&presult.Template)
	templateNames := TemplateNames(s.fileContents)
	list.Items = make([]lsp.CompletionItem, len(baseCompletionItems), len(baseCompletionItems)+len(tParams)+len(templateNames))
	list.Items = append(list.Items, baseCompletionItems...)
	for _, param := range tParams {
		segments := strings.Split(param, " ")
		list.Items = append(list.Items, lsp.CompletionItem{
			Label:            segments[0],
			InsertText:       segments[0],
			InsertTextMode:   lsp.InsertTextModeAsIs,
			InsertTextFormat: lsp.InsertTextFormatPlainText,
			Kind:             lsp.CompletionItemKindVariable,
		})
	}
	for _, template := range templateNames {
		list.Items = append(list.Items, lsp.CompletionItem{
			Label:            template.name,
			InsertText:       fmt.Sprintf("%s($1)", template.name),
			InsertTextMode:   lsp.InsertTextModeAsIs,
			InsertTextFormat: lsp.InsertTextFormatSnippet,
			Kind:             lsp.CompletionItemKindFunction,
		})
	}
	return &list, nil
}

func (s *GwirlLspServer) CompletionResolve(ctx context.Context, params *lsp.CompletionItem) (result *lsp.CompletionItem, err error) {
	return nil, nil
}

func (s *GwirlLspServer) Declaration(ctx context.Context, params *lsp.DeclarationParams) (result []lsp.Location, err error) {
	return nil, nil
}

func (s *GwirlLspServer) Definition(ctx context.Context, params *lsp.DefinitionParams) (result []lsp.Location, err error) {
	s.logf("Definition params=%v", params)
	contents := s.fileContents[params.TextDocument.URI]
	if contents == "" {
		return []lsp.Location{}, nil
	}
	var definition string
	res := s.parser.Parse(contents, params.TextDocument.URI.Filename())
	tt := FindTemplateTreeForPosition(res.Template.Content, LspPosToParserPos(params.Position))
	if tt == nil {
		s.log("Didn't find a match")
		return []lsp.Location{}, nil
	}

	switch tt.Type {
	case parser.TT2GoExp:
		// Go Exp code is single line, find the segment where the
		// lsp.Position.Character belongs.
		// For now, we can only really resolve the first segment since we don't
		// have any type information.
		segments := splitOnSet(tt.Text, ".()")
		if len(segments) > 0 && params.Position.Character <= uint32(tt.Column()+len(segments[0])) {
			definition = segments[0]
		}
	}

	paramNames := GetTemplateParamNames(&res.Template)
	locs := []lsp.Location{}
	for _, param := range paramNames {
		if definition == param {
			startLine := res.Template.Params.Line()
			// Adding one to the index to account for dropping the paren
			startCol := strings.Index(res.Template.Params.Str, param) + 1
			loc := lsp.Location{
				URI: params.TextDocument.URI,
				Range: lsp.Range{
					Start: ParserPosToLspPos(PlainPosition{
						line:   startLine,
						column: startCol,
					}),
					End: ParserPosToLspPos(PlainPosition{
						line:   startLine,
						column: startCol + len(param),
					}),
				},
			}
			locs = append(locs, loc)
		}
	}
	for fileUri := range s.fileContents {
		templateName := TemplateName(fileUri.Filename())
		if templateName == definition {
			beginning := ParserPosToLspPos(PlainPosition{line: 1, column: 0})
			loc := lsp.Location{
				URI: fileUri,
				Range: lsp.Range{
					Start: beginning,
					End:   beginning,
				},
			}
			locs = append(locs, loc)
		}
	}
	return locs, nil
}

func (s *GwirlLspServer) DidChange(ctx context.Context, params *lsp.DidChangeTextDocumentParams) (err error) {
	s.logf("DidChange params=%v", params)
	s.openedFiles = append(s.openedFiles, params.TextDocument.URI.Filename())
	if len(params.ContentChanges) > 0 {
		s.fileContents[params.TextDocument.URI] = params.ContentChanges[0].Text
	}

	res := s.parser.Parse(s.fileContents[params.TextDocument.URI], params.TextDocument.URI.Filename())

	diagnostics := make([]lsp.Diagnostic, len(res.Errors), len(res.Errors))
	for i, e := range res.Errors {
		s.logf("error: %v", e)
		line := uint32(e.Line())
		column := uint32(e.Column())
		d := lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{Line: line, Character: column},
				End:   lsp.Position{Line: line, Character: column},
			},
			Severity: lsp.DiagnosticSeverityError,
			Source:   "gwirl-lsp",
			Message:  e.String(),
		}
		diagnostics[i] = d
	}

    s.sendNotification("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
        Diagnostics: diagnostics,
        URI:         params.TextDocument.URI,
    })

	return nil
}

func (s *GwirlLspServer) DidChangeConfiguration(ctx context.Context, params *lsp.DidChangeConfigurationParams) (err error) {
	return errors.New("Not implemented")
}

func (s *GwirlLspServer) DidChangeWatchedFiles(ctx context.Context, params *lsp.DidChangeWatchedFilesParams) (err error) {
	return errors.New("Not implemented")
}

func (s *GwirlLspServer) DidChangeWorkspaceFolders(ctx context.Context, params *lsp.DidChangeWorkspaceFoldersParams) (err error) {
	return errors.New("Not implemented")
}

func (s *GwirlLspServer) DidClose(ctx context.Context, params *lsp.DidCloseTextDocumentParams) (err error) {
	return errors.New("Not implemented")
}

func (s *GwirlLspServer) DidOpen(ctx context.Context, params *lsp.DidOpenTextDocumentParams) (err error) {
	s.logf("DidOpen params=%v", params)
	s.openedFiles = append(s.openedFiles, params.TextDocument.URI.Filename())
	s.fileContents[params.TextDocument.URI] = params.TextDocument.Text

	res := s.parser.Parse(params.TextDocument.Text, params.TextDocument.URI.Filename())

	diagnostics := make([]lsp.Diagnostic, len(res.Errors), len(res.Errors))
	for i, e := range res.Errors {
		line := uint32(e.Line())
		column := uint32(e.Column())
		d := lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{Line: line, Character: column},
				End:   lsp.Position{Line: line, Character: column},
			},
			Severity: lsp.DiagnosticSeverityError,
			Source:   "gwirl-lsp",
			Message:  e.String(),
		}
		diagnostics[i] = d
	}

	s.sendNotification("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
		Diagnostics: diagnostics,
		URI:         params.TextDocument.URI,
	})
	return nil
}
func (s *GwirlLspServer) DidSave(ctx context.Context, params *lsp.DidSaveTextDocumentParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) DocumentColor(ctx context.Context, params *lsp.DocumentColorParams) (result []lsp.ColorInformation, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DocumentHighlight(ctx context.Context, params *lsp.DocumentHighlightParams) (result []lsp.DocumentHighlight, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DocumentLink(ctx context.Context, params *lsp.DocumentLinkParams) (result []lsp.DocumentLink, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DocumentLinkResolve(ctx context.Context, params *lsp.DocumentLink) (result *lsp.DocumentLink, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DocumentSymbol(ctx context.Context, params *lsp.DocumentSymbolParams) (result []interface{}, err error) {
	return nil, nil
}
func (s *GwirlLspServer) ExecuteCommand(ctx context.Context, params *lsp.ExecuteCommandParams) (result interface{}, err error) {
	return nil, nil
}
func (s *GwirlLspServer) FoldingRanges(ctx context.Context, params *lsp.FoldingRangeParams) (result []lsp.FoldingRange, err error) {
	return nil, nil
}
func (s *GwirlLspServer) Formatting(ctx context.Context, params *lsp.DocumentFormattingParams) (result []lsp.TextEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) Hover(ctx context.Context, params *lsp.HoverParams) (result *lsp.Hover, err error) {
	contents := s.fileContents[params.TextDocument.URI]
	if contents == "" {
		return nil, nil
	}
	p := parser.NewParser2("")
	res := p.Parse(contents, params.TextDocument.URI.Filename())
	word := FindWordAtPosition(&res.Template, params.Position)
	if word == "" {
		s.log("Didn't find a word for hover")
		return nil, nil
	}
	hover := &lsp.Hover{
		Contents: lsp.MarkupContent{
			Kind:  "markdown",
			Value: word,
		},
	}
	return hover, nil
}
func (s *GwirlLspServer) Implementation(ctx context.Context, params *lsp.ImplementationParams) (result []lsp.Location, err error) {
	return nil, nil
}
func (s *GwirlLspServer) OnTypeFormatting(ctx context.Context, params *lsp.DocumentOnTypeFormattingParams) (result []lsp.TextEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) PrepareRename(ctx context.Context, params *lsp.PrepareRenameParams) (result *lsp.Range, err error) {
	return nil, nil
}
func (s *GwirlLspServer) RangeFormatting(ctx context.Context, params *lsp.DocumentRangeFormattingParams) (result []lsp.TextEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) References(ctx context.Context, params *lsp.ReferenceParams) (result []lsp.Location, err error) {
	return nil, nil
}
func (s *GwirlLspServer) Rename(ctx context.Context, params *lsp.RenameParams) (result *lsp.WorkspaceEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) SignatureHelp(ctx context.Context, params *lsp.SignatureHelpParams) (result *lsp.SignatureHelp, err error) {
	return nil, nil
}
func (s *GwirlLspServer) Symbols(ctx context.Context, params *lsp.WorkspaceSymbolParams) (result []lsp.SymbolInformation, err error) {
	return nil, nil
}
func (s *GwirlLspServer) TypeDefinition(ctx context.Context, params *lsp.TypeDefinitionParams) (result []lsp.Location, err error) {
	return nil, nil
}
func (s *GwirlLspServer) WillSave(ctx context.Context, params *lsp.WillSaveTextDocumentParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) WillSaveWaitUntil(ctx context.Context, params *lsp.WillSaveTextDocumentParams) (result []lsp.TextEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) ShowDocument(ctx context.Context, params *lsp.ShowDocumentParams) (result *lsp.ShowDocumentResult, err error) {
	return nil, nil
}
func (s *GwirlLspServer) WillCreateFiles(ctx context.Context, params *lsp.CreateFilesParams) (result *lsp.WorkspaceEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DidCreateFiles(ctx context.Context, params *lsp.CreateFilesParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) WillRenameFiles(ctx context.Context, params *lsp.RenameFilesParams) (result *lsp.WorkspaceEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DidRenameFiles(ctx context.Context, params *lsp.RenameFilesParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) WillDeleteFiles(ctx context.Context, params *lsp.DeleteFilesParams) (result *lsp.WorkspaceEdit, err error) {
	return nil, nil
}
func (s *GwirlLspServer) DidDeleteFiles(ctx context.Context, params *lsp.DeleteFilesParams) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) CodeLensRefresh(ctx context.Context) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) PrepareCallHierarchy(ctx context.Context, params *lsp.CallHierarchyPrepareParams) (result []lsp.CallHierarchyItem, err error) {
	return nil, nil
}
func (s *GwirlLspServer) IncomingCalls(ctx context.Context, params *lsp.CallHierarchyIncomingCallsParams) (result []lsp.CallHierarchyIncomingCall, err error) {
	return nil, nil
}
func (s *GwirlLspServer) OutgoingCalls(ctx context.Context, params *lsp.CallHierarchyOutgoingCallsParams) (result []lsp.CallHierarchyOutgoingCall, err error) {
	return nil, nil
}
func (s *GwirlLspServer) SemanticTokensFull(ctx context.Context, params *lsp.SemanticTokensParams) (result *lsp.SemanticTokens, err error) {
	contents := s.fileContents[params.TextDocument.URI]
	if contents == "" {
		return nil, nil
	}

	res := s.parser.Parse(contents, params.TextDocument.URI.Filename())
	tokens := CreateTemplateSemanticTokens(&res.Template)
	return tokens, nil
}
func (s *GwirlLspServer) SemanticTokensFullDelta(ctx context.Context, params *lsp.SemanticTokensDeltaParams) (result interface{}, err error) {
	return nil, nil
}
func (s *GwirlLspServer) SemanticTokensRange(ctx context.Context, params *lsp.SemanticTokensRangeParams) (result *lsp.SemanticTokens, err error) {
	return nil, nil
}
func (s *GwirlLspServer) SemanticTokensRefresh(ctx context.Context) (err error) {
	return errors.New("Not implemented")
}
func (s *GwirlLspServer) LinkedEditingRange(ctx context.Context, params *lsp.LinkedEditingRangeParams) (result *lsp.LinkedEditingRanges, err error) {
	return nil, nil
}
func (s *GwirlLspServer) Moniker(ctx context.Context, params *lsp.MonikerParams) (result []lsp.Moniker, err error) {
	return nil, nil
}
func (s *GwirlLspServer) Request(ctx context.Context, method string, params interface{}) (result interface{}, err error) {
	return nil, nil
}
