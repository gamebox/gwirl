package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gamebox/gwirl/internal/gen"
	"github.com/gamebox/gwirl/internal/parser"
	"github.com/gamebox/gwirl/internal/pathutils"
	"go.lsp.dev/jsonrpc2"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var pathConfig pathutils.PathConfig

func init() {
    pathConfig = pathutils.PathConfig{
        TemplateBase: "templates",
        DestBase: "views",
    }
}

func handleResponse[I any](s *GwirlLspServer, method string, handler func(context.Context, *I) error, ctx context.Context, bytes []byte) {
    var response jsonrpc2.Response
    err := json.Unmarshal(bytes, &response)
    if err != nil {
        s.logf(`Method "%s" failed to unmarshal request: %e`, method, err)
        return
    }
    var params I
    err = json.Unmarshal(response.Result(), &params)
    if err != nil || response.Err() != nil {
        return
    }
    err = handler(ctx, &params)
    if err != nil {
        return
    }
}

func handleRequest[I any, O any](s *GwirlLspServer, request *jsonrpc2.Call, handler func(context.Context, *I) (O, error), ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			s.sendNotification(lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
				Message: fmt.Sprintf("Server crashed with error: %v", err),
				Type:    lsp.MessageTypeInfo,
			})
		}
	}()
	var params I
    err := json.Unmarshal(request.Params(), &params)
    method := request.Method()
	if err != nil {
		s.logf(`Method "%s" failed to unmarshal params: %e`, method, err)
		e := jsonrpc2.NewError(jsonrpc2.ParseError, fmt.Sprintf("Error handling method \"%s\": %v", method, err))
        msg, err := jsonrpc2.NewResponse(request.ID(), nil, e)
        if err != nil {
            s.logf("Failed to create jsonrpc2 response for request %d", request.ID())
        }
		s.host.WriteMessage(msg)
		return
	}
	result, err := handler(ctx, &params)
	if err != nil {
		s.logf(`Method "%s" handler returned error: %e`, method, err)
		e := jsonrpc2.NewError(jsonrpc2.ParseError, fmt.Sprintf("Error handling method \"%s\": %v", method, err))
        msg, err := jsonrpc2.NewResponse(request.ID(), nil, e)
        if err != nil {
            s.logf("Failed to create jsonrpc2 response for request %d", request.ID())
        }
		s.host.WriteMessage(msg)
		return
	}
	s.logf(`Method "%s" responding successfully`, method)
    msg, err := jsonrpc2.NewResponse(
        request.ID(),
        result,
        nil,
    )
    if err != nil {
        return
    }
	s.host.WriteMessage(msg)
}

func handleNotification[I any](s *GwirlLspServer, request *jsonrpc2.Notification, handler func(context.Context, *I) error, ctx context.Context) {
	var params I
    err := json.Unmarshal(request.Params(), &params)
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
	logWriter    io.Writer
	logMutex     *sync.Mutex
	openedFiles  []string
    templateCache *TemplateCache
	context      context.Context
	parser       *parser.Parser2
    gopls        *LspMsgRW
    host         *LspMsgRW
    goplscmd     *exec.Cmd
    nextReqId    atomic.Uint32
    pending      map[jsonrpc2.ID]chan *jsonrpc2.Response
    pendingMutex *sync.Mutex
}

func NewGwirlLspServer(writer io.Writer, logWriter io.Writer, reader io.Reader, context context.Context) *GwirlLspServer {
	log.SetOutput(logWriter)
	p := parser.NewParser2("")
    goplscmd := exec.Command("/Users/anthonybullard/.local/share/nvim/mason/bin/gopls", "-verbose", "-logfile=/Users/anthonybullard/Library/Caches/gopls/gopls.log.txt")
    goplsstdin, err := goplscmd.StdoutPipe()
    goplsstdout, err := goplscmd.StdinPipe()
    err = goplscmd.Start()
    pending := map[jsonrpc2.ID]chan *jsonrpc2.Response{}
    templateCache := NewTemplateCache()
	s := GwirlLspServer{
		logWriter:    logWriter,
		logMutex:     &sync.Mutex{},
		openedFiles:  make([]string, 10, 100),
        templateCache: &templateCache,
		context:      context,
		parser:       &p,
        goplscmd:     goplscmd,
        nextReqId:    atomic.Uint32{},
        pending:      pending,
        pendingMutex: &sync.Mutex{},
	}
    if err != nil {
        s.logf("Could not start gopls: %v", err)
    }
    gopls, err := NewMsgRWFromIO(goplsstdin, goplsstdout)
    if err != nil {
        s.logf("Could not create readwriter for gopls: %v", err)
        s.gopls = nil
    } else {
        s.gopls = &gopls
    }
    host, err := NewMsgRWFromIO(reader, writer)
    if err != nil {
        s.host = nil
    } else {
        s.host = &host
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
    wg := sync.WaitGroup{}
    wg.Add(1)
    go s.handleClientRequests(&wg)
    if s.gopls != nil && s.goplscmd.Process != nil {
        s.logf("Started gopls with pid %d", s.goplscmd.Process.Pid)
        wg.Add(1)
        go s.handleGoPls(&wg)
    }
    wg.Wait()
}

func (s *GwirlLspServer) handleGoPls(wg *sync.WaitGroup) {
    defer wg.Done()
    for {
        if s.goplscmd.Process == nil {
            s.logf("gopls has exited :-(")
            return
        }
        s.log("Waiting for message from gopls")
        msg, err := s.gopls.ReadMessage()
        if err != nil {
            s.logf("Failed to read gopls message: %v", err)
            continue
        }
        switch msg := msg.(type) {
        case *jsonrpc2.Response:
			s.log("==========================================")
			s.logf("Received gopls response for request \"%d\"", msg.ID())
			s.log("------------------------------------------")
            s.pendingMutex.Lock()
            c, ok := s.pending[msg.ID()]
            defer s.pendingMutex.Unlock()
            if ok {
                delete(s.pending, msg.ID())
                c<-msg
            }
            return
        case *jsonrpc2.Notification:
			s.log("==========================================")
			s.logf("Received gopls notification for method \"%s\"", msg.Method())
			s.log("------------------------------------------")
        case *jsonrpc2.Call:
            s.log("Ignoring request from gopls")
            return
        }
    }
}

func (s *GwirlLspServer) handleClientRequests(wg *sync.WaitGroup) {
    defer wg.Done()
	for {
		msg, err := s.host.ReadMessage()
		if err != nil {
            s.logf("Error occurred reading a message: %v", err)
			return
		}
        switch msg := msg.(type) {
        case *jsonrpc2.Call:
            method := msg.Method()
			s.log("==========================================")
			s.logf("Received client call for method \"%s\"", msg.Method())
			s.log("------------------------------------------")
            switch method {
            case lsp.MethodInitialize:
                go handleRequest[lsp.InitializeParams, *lsp.InitializeResult](s, msg, s.Initialize, s.context)
            case lsp.MethodTextDocumentCompletion:
                go handleRequest[lsp.CompletionParams, *lsp.CompletionList](s, msg, s.Completion, s.context)
            case lsp.MethodTextDocumentDefinition:
                go handleRequest[lsp.DefinitionParams, []lsp.Location](s, msg, s.Definition, s.context)
            case lsp.MethodSemanticTokensFull:
                go handleRequest[lsp.SemanticTokensParams, *lsp.SemanticTokens](s, msg, s.SemanticTokensFull, s.context)
            case lsp.MethodTextDocumentHover:
                go handleRequest[lsp.HoverParams, *lsp.Hover](s, msg, s.Hover, s.context)
            case lsp.MethodShutdown:
                s.log("Shutting down...")
                s.sendGoPlsNotification(lsp.MethodShutdown, struct{}{})
                return
            default:
            }
        case *jsonrpc2.Notification:
            method := msg.Method()
			s.log("==========================================")
			s.logf("Received client notification for method \"%s\"", msg.Method())
			s.log("------------------------------------------")
            switch method {
            case lsp.MethodInitialized:
                go handleNotification[lsp.InitializedParams](s, msg, s.Initialized, s.context)
            case lsp.MethodTextDocumentDidOpen:
                go handleNotification[lsp.DidOpenTextDocumentParams](s, msg, s.DidOpen, s.context)
            case lsp.MethodTextDocumentDidChange:
                go handleNotification[lsp.DidChangeTextDocumentParams](s, msg, s.DidChange, s.context)
            }
        case *jsonrpc2.Response:
            s.log("Ignoring response from client")
        }
	}
}

func (s *GwirlLspServer) getReqId() uint32 {
    id := s.nextReqId.Add(1)
    return id
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

func (s *GwirlLspServer) Write(bytes []byte) (n int, err error) {
    s.logMutex.Lock()
    defer s.logMutex.Unlock()
    log.Println("Writing raw output on lines below\n-----------------")
    n, err = log.Writer().Write(bytes)
    log.Println("------------Raw output complete-------------")
    return
}

func (s *GwirlLspServer) sendNotification(method string, notification any) {
	if s.host == nil {
        return
	}
    n, err := jsonrpc2.NewNotification(
        method,
        notification,
    )
    if err != nil {
        s.logf("Could not send message: %v", notification)
        return
    }
    s.log("==========================================")
    s.logf("Sending client notification for method \"%s\"", n.Method())
    notiJson, err := json.Marshal(notification)
    s.logf("%s\n", notiJson)
    s.log("------------------------------------------")
    s.host.WriteMessage(n)
}

func (s *GwirlLspServer) sendGoPlsRequest(method string, req any) (*jsonrpc2.Response, error) {
    if s.gopls == nil {
        s.logf("Cannot send %s request to gopls: gopls not started ", method)
        return nil, errors.New("gopls not started")
    }

    reqId := jsonrpc2.NewNumberID(int32(s.getReqId()))
    request, err := jsonrpc2.NewCall(
        reqId,
        method,
        req,
    )

    s.log("==========================================")
    s.logf("Sending gopls request for method \"%s\"", method)
    notiJson, err := json.Marshal(req)
    s.logf("%s\n", notiJson)
    s.log("------------------------------------------")

    err = s.gopls.WriteMessage(request)
    if err != nil {
        s.logf("Could not write %s request to gopls: %v", method, err)
        return nil, errors.New("gopls write error")
    }

    responseChan := make(chan *jsonrpc2.Response, 1)
    timeout, cancel := context.WithTimeout(s.context, 5 * time.Second)
    defer cancel()

    s.pendingMutex.Lock()
    s.pending[reqId] = responseChan
    s.pendingMutex.Unlock()

    defer func() {
        s.pendingMutex.Lock()
        delete(s.pending, reqId)
        s.pendingMutex.Unlock()
    }()

    select {
        case response := <-responseChan:
            return response, nil
        case <- timeout.Done():
            return nil, errors.New("response timeout")
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
			contents, err := os.ReadFile(entries[idx])
			if err != nil {
				continue
			}
            var cacheItem TemplateCacheItem
            result := s.parser.Parse(string(contents), entry.Name(), entry.Name())
            if len(result.Errors) > 0 {
                return
            }
            cacheItem.text = string(contents)
            cacheItem.template = result.Template
            sb := strings.Builder{}
            g := gen.NewGenerator(false)
            g.Generate(cacheItem.template, "html", )
			s.fileContents[uri.File(entries[idx])] = string(contents)
		}
	}
	s.logf("Loaded %d templates", len(s.fileContents))
}

func (s *GwirlLspServer) sendGoPlsNotification(method string, params any) error {
    notification, err := jsonrpc2.NewNotification(method, params)
    if err != nil {
        s.logf("Error sending gopls notification for method %s: %v", method, err)
        return err
    }
    s.log("==========================================")
    s.logf("Sending gopls notification for method \"%s\"", method)
    s.log("------------------------------------------")
    s.gopls.WriteMessage(notification)
    return nil
}

func (s *GwirlLspServer) Initialize(ctx context.Context, params *lsp.InitializeParams) (result *lsp.InitializeResult, err error) {
	s.logf("Initializing with rootURI \"%s\" and rootPath \"%s\"", params.RootURI.Filename(), params.RootPath)
	go s.RetrieveWorkspaceTemplates(params.RootURI.Filename())
    response, err := s.sendGoPlsRequest(lsp.MethodInitialize, lsp.InitializeParams{
       ProcessID: int32(os.Getpid()),
       ClientInfo: &lsp.ClientInfo{
           Name: "gwirl-lsp",
           Version: "0.0.1",
       },
       Locale: params.Locale,
       WorkspaceFolders: params.WorkspaceFolders,
       RootPath: params.RootPath,
       WorkDoneProgressParams: params.WorkDoneProgressParams,
       InitializationOptions: params.InitializationOptions,
       RootURI: params.RootURI,
       Trace: params.Trace,
       Capabilities: params.Capabilities,
    })
    if err == nil {
        s.sendGoPlsNotification(lsp.MethodInitialized, lsp.InitializedParams{}) 
    } else {
        s.logf("Received error from gopls during initialize: %v", err)
    }

    var supportsSemanticHighlighting bool
    var res *lsp.InitializeResult
    err = json.Unmarshal(response.Result(), res)
    if err == nil {
        supportsSemanticHighlighting = res.Capabilities.SemanticTokensProvider != nil
    }

    if supportsSemanticHighlighting {
        s.log("GOPLS SUPPORTS SEMANTIC HIGHLIGHTING")
    }

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
	presult := s.parser.Parse(s.fileContents[params.TextDocument.URI], params.TextDocument.URI.Filename(), params.TextDocument.URI.Filename())
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
    response, err := s.sendGoPlsRequest(lsp.MethodTextDocumentCompletion, lsp.CompletionParams{
        TextDocumentPositionParams: lsp.TextDocumentPositionParams{
            TextDocument: lsp.TextDocumentIdentifier{
                URI: uri.File(pathConfig.GoPath(params.TextDocument.URI.Filename())),
            },
            Position: params.Position,
        },
        PartialResultParams: params.PartialResultParams,
    })
    if err != nil {
        return &list, nil
    }

    var goplsCompletions *lsp.CompletionList
    err = json.Unmarshal(response.Result(), goplsCompletions)
    if err != nil {
        return &list, nil
    }

    list.Items = append(list.Items, goplsCompletions.Items...)

	return &list, nil
}

func (s *GwirlLspServer) CompletionResolve(ctx context.Context, params *lsp.CompletionItem) (result *lsp.CompletionItem, err error) {
	return nil, nil
}

func (s *GwirlLspServer) Declaration(ctx context.Context, params *lsp.DeclarationParams) (result []lsp.Location, err error) {
	return nil, nil
}

func (s *GwirlLspServer) Definition(ctx context.Context, params *lsp.DefinitionParams) (result []lsp.Location, err error) {
	contents := s.fileContents[params.TextDocument.URI]
	if contents == "" {
		return []lsp.Location{}, nil
	}
	var definition string
	res := s.parser.Parse(contents, params.TextDocument.URI.Filename(), params.TextDocument.URI.Filename())
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
	s.openedFiles = append(s.openedFiles, params.TextDocument.URI.Filename())
	if len(params.ContentChanges) > 0 {
		s.fileContents[params.TextDocument.URI] = params.ContentChanges[0].Text
	}

	res := s.parser.Parse(s.fileContents[params.TextDocument.URI], params.TextDocument.URI.Filename(), params.TextDocument.URI.Filename())

	diagnostics := createDiagnosticsFromErrors(res.Errors)

	s.sendNotification("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
		Diagnostics: diagnostics,
		URI:         params.TextDocument.URI,
	})

    if len(res.Errors) > 0 {
        return nil
    }

    g := gen.NewGenerator(false)
    sb := strings.Builder{}
    err = g.Generate(res.Template, "html", &sb)
    if err != nil {
        s.logf("Failed to generate new go file for %s", params.TextDocument.URI.Filename())
        return nil
    }
    err = s.sendGoPlsNotification(lsp.MethodTextDocumentDidChange, lsp.DidChangeTextDocumentParams{
        TextDocument: lsp.VersionedTextDocumentIdentifier{
            Version: params.TextDocument.Version,
            TextDocumentIdentifier: lsp.TextDocumentIdentifier{
                URI: uri.File(pathConfig.GoPath(params.TextDocument.URI.Filename())),
            },
        },
        ContentChanges: []lsp.TextDocumentContentChangeEvent{
            { Text: sb.String(), Range: params.ContentChanges[0].Range, RangeLength: params.ContentChanges[0].RangeLength },
        },
    })
    if err != nil {
        s.logf("Failed to send notification to gopls")
        return nil
    }

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
	s.openedFiles = append(s.openedFiles, params.TextDocument.URI.Filename())
	s.fileContents = params.TextDocument.Text

	res := s.parser.Parse(params.TextDocument.Text, params.TextDocument.URI.Filename(), params.TextDocument.URI.Filename())

	diagnostics := createDiagnosticsFromErrors(res.Errors)

	s.sendNotification("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
		Diagnostics: diagnostics,
		URI:         params.TextDocument.URI,
	})
        
    if len(res.Errors) > 0 {
        return nil
    }

    g := gen.NewGenerator(false)
    sb := strings.Builder{}
    err = g.Generate(res.Template, "html", &sb)
    if err != nil {
        s.logf("Failed to generate new go file for %s", params.TextDocument.URI.Filename())
        return nil
    }
    err = s.sendGoPlsNotification(lsp.MethodTextDocumentDidChange, lsp.DidOpenTextDocumentParams{
        TextDocument: params.TextDocument,
    })
    if err != nil {
        s.logf("Failed to send notification to gopls")
        return nil
    }
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
    res, err := s.sendGoPlsRequest(lsp.MethodTextDocumentHover, lsp.HoverParams{
        TextDocumentPositionParams: lsp.TextDocumentPositionParams{
            TextDocument: lsp.TextDocumentIdentifier{
                URI: uri.File(pathConfig.GoPath(params.TextDocument.URI.Filename())),
            },
            Position: params.Position,
        },
    })

    if err != nil {
        s.logf("%s error sending gopls request: %v", lsp.MethodTextDocumentHover, err)
        return nil, nil
    }

    if res.Err() != nil {
        s.logf("%s error in gopls response: %v", lsp.MethodTextDocumentHover, res.Err())
        return nil, res.Err()
    }

    s.logf("Received gopls result: %s", res.Result())

    var hover *lsp.Hover
    err = json.Unmarshal(res.Result(), hover)
    if err != nil {
        s.logf("%s error unmarshalling gopls request: %v", lsp.MethodTextDocumentHover, err)
        return nil, nil
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
	contents, ok := s.templateCache.Load(params.TextDocument.URI)
	if ok {
		return nil, nil
	}
    cached, ok := contents.(TemplateCacheItem)

	tokens := CreateTemplateSemanticTokens(&cached.template)
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

func createDiagnosticsFromErrors(errors []parser.ParseError) []lsp.Diagnostic {
	diagnostics := make([]lsp.Diagnostic, len(errors), len(errors))
	for i, e := range errors {
		d := lsp.Diagnostic{
			Range: lsp.Range{
				Start: ParserPosToLspPos(e.Start),
				End:   ParserPosToLspPos(e.End),
			},
			Severity: lsp.DiagnosticSeverityError,
			Source:   "gwirl-lsp",
			Message:  e.Err,
		}
		diagnostics[i] = d
	}
	return diagnostics
}
