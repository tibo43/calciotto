package handlers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// uncodedMessages are the fixed messages that deliberately carry no code: a
// server-side failure, a request only a script or a broken client can send,
// or a check the app never lets a person reach. They stay in English. Any
// other message passed to messageBody (or set as a groupIDFailure message)
// must have a code in messageCodes, or be the text of a coded service error,
// so a reworded handler message can't silently lose its translation.
var uncodedMessages = map[string]bool{
	// A malformed request only a script or a broken client sends: a bad id
	// in the path or the query, a body of the wrong shape or type.
	"expected application/json":                      true,
	"invalid group id":                               true,
	"invalid group_id":                               true,
	"invalid match id":                               true,
	"invalid player id":                              true,
	"invalid team id":                                true,
	"invalid passkey ID":                             true,
	"invalid passkey request":                        true,
	"invalid request body":                           true,
	"group_id does not match the authorized group":   true,
	"PreferredPositions is required":                 true,
	"PreferredPositions must be a list of positions": true,
	"locale is required (a language code, or null to follow the phone)": true,
	"locale must be a language code or null":                            true,
	"max_players is required":                                           true,
	"preferences is required":                                           true,
	"voted_for_id is required":                                          true,

	// Authentication and session plumbing: the app reacts to the status (it
	// logs the player out, or renews and retries) and never shows the text.
	"missing authentication":                  true,
	"missing or invalid Authorization header": true,
	"unauthorized":                            true,
	"invalid session request origin":          true,
	"missing or invalid session ID":           true,
	"the active session changed":              true,

	// Server-side failures: a lost database, a route wired without its
	// middleware. Nothing the player can act on.
	"group not resolved":                                  true,
	"group not resolved for this match":                   true,
	"route is missing its group authorization middleware": true,
	"could not load guide status":                         true,
	"could not save guide completion":                     true,
	"unable to load groups":                               true,
	"unable to load matches":                              true,
	"unable to end the session":                           true,
}

// handlerMessages lists every string literal the package's handler files pass
// to messageBody, or set as the message of a groupIDFailure (written through
// messageBody later), with where it appears.
func handlerMessages(t *testing.T) map[string][]string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string][]string{}
	fset := token.NewFileSet()
	record := func(expr ast.Expr) {
		lit, ok := expr.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return
		}
		text, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Fatalf("%s: %v", fset.Position(lit.Pos()), err)
		}
		found[text] = append(found[text], fset.Position(lit.Pos()).String())
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				if fn, ok := node.Fun.(*ast.Ident); ok && fn.Name == "messageBody" && len(node.Args) == 1 {
					record(node.Args[0])
				}
			case *ast.CompositeLit:
				if typ, ok := node.Type.(*ast.Ident); ok && typ.Name == "groupIDFailure" {
					for _, elt := range node.Elts {
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "message" {
								record(kv.Value)
							}
						}
					}
				}
			}
			return true
		})
	}
	return found
}

func TestHandlerMessages_HaveACodeOrAreListedAsUncoded(t *testing.T) {
	found := handlerMessages(t)
	if len(found) == 0 {
		t.Fatal("no messageBody call found: the scan is broken")
	}
	var missing []string
	for message, where := range found {
		_, coded := messageBody(message)["code"]
		switch {
		case coded && uncodedMessages[message]:
			t.Errorf("%q has a code: remove it from uncodedMessages", message)
		case !coded && !uncodedMessages[message]:
			missing = append(missing, strconv.Quote(message)+" ("+strings.Join(where, ", ")+")")
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("message without a code: %s; add it to messageCodes (and the locale bundles), or to uncodedMessages if no person can see it", m)
	}
	for message := range uncodedMessages {
		if _, used := found[message]; !used {
			t.Errorf("%q is in uncodedMessages but no handler sends it any more: remove it", message)
		}
	}
	for message := range messageCodes {
		if _, used := found[message]; !used {
			t.Errorf("%q is in messageCodes but no handler sends it any more", message)
		}
	}
}
