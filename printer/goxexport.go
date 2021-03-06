package printer

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/8byt/gox/ast"
	"github.com/8byt/gox/token"
)

// Map html-style to actual js event names
var eventMap = map[string]string{
	"onAbort":          "abort",
	"onCancel":         "cancel",
	"onCanPlay":        "canplay",
	"onCanPlaythrough": "canplaythrough",
	"onChange":         "change",
	"onClick":          "click",
	"onCueChange":      "cuechange",
	"onDblClick":       "dblclick",
	"onDurationChange": "durationchange",
	"onEmptied":        "emptied",
	"onEnded":          "ended",
	"onInput":          "input",
	"onInvalid":        "invalid",
	"onKeyDown":        "keydown",
	"onKeyPress":       "keypress",
	"onKeyUp":          "keyup",
	"onLoadedData":     "loadeddata",
	"onLoadedMetadata": "loadedmetadata",
	"onLoadStart":      "loadstart",
	"onMouseDown":      "mousedown",
	"onMouseEnter":     "mouseenter",
	"onMouseleave":     "mouseleave",
	"onMouseMove":      "mousemove",
	"onMouseOut":       "mouseout",
	"onMouseOver":      "mouseover",
	"onMouseUp":        "mouseup",
	"onMouseWheel":     "mousewheel",
	"onPause":          "pause",
	"onPlay":           "play",
	"onPlaying":        "playing",
	"onProgress":       "progress",
	"onRateChange":     "ratechange",
	"onReset":          "reset",
	"onSeeked":         "seeked",
	"onSeeking":        "seeking",
	"onSelect":         "select",
	"onShow":           "show",
	"onStalled":        "stalled",
	"onSubmit":         "submit",
	"onSuspend":        "suspend",
	"onTimeUpdate":     "timeupdate",
	"onToggle":         "toggle",
	"onVolumeChange":   "volumechange",
	"onWaiting":        "waiting",
}

var attrMap = map[string]string{
	"autofocus":   "autofocus",
	"checked":     "checked",
	"class":       "class",
	"for":         "htmlFor",
	"href":        "href",
	"id":          "id",
	"placeholder": "placeholder",
	"src":         "src",
	"type":        "type",
	"value":       "value",
}

func goxToVecty(genname string, gox *ast.GoxExpr) ast.Expr {
	isComponent := true
	if t, ok := gox.TagName.(*ast.Ident); ok {
		isComponent = unicode.IsUpper(rune(t.Name[0]))
	}

	if isComponent {
		return newComponent(genname, gox)
	} else {
		args := []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(gox.TagName.(*ast.Ident).Name),
			}}

		if len(gox.Attrs) > 0 {
			// Create markup expr and add attributes
			markup := newCallExpr(
				newSelectorExpr(genname, "Markup"),
				mapProps(genname, gox.Attrs),
			)

			// Add the markup
			args = append(args, markup)
		}

		// Add the contents
		for _, expr := range gox.X {
			switch expr := expr.(type) {
			// TODO figure out what's a better thing to do here
			// do we want to error on compile or figure out what to do based on context?
			// (I think the latter)
			// Fallback to regular behavior, don't wrap this yet
			//case *ast.GoExpr:
			//	e := newCallExpr(
			//		newSelectorExpr(genname, "Text"),
			//		[]ast.Expr{expr},
			//	)
			//	args = append(args, e)

			case *ast.BareWordsExpr:
				if len(strings.TrimSpace(expr.Value)) == 0 {
					continue
				}
				e := newCallExpr(
					newSelectorExpr(genname, "Text"),
					[]ast.Expr{expr},
				)
				args = append(args, e)
			case *ast.GoExpr:
				e := newCallExpr(
					newSelectorExpr(genname, "Value"),
					[]ast.Expr{expr},
				)
				args = append(args, e)
			default:
				args = append(args, expr)
			}
		}

		return newCallExpr(
			newSelectorExpr(genname, "Tag"),
			args,
		)
	}
}

func newSelectorExpr(x, sel string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   ast.NewIdent(x),
		Sel: ast.NewIdent(sel)}
}

func newCallExpr(fun ast.Expr, args []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:      fun,
		Args:     args,
		Ellipsis: token.NoPos, Lparen: token.NoPos, Rparen: token.NoPos}
}

func newComponent(genname string, gox *ast.GoxExpr) ast.Expr {
	args := make([]ast.Expr, len(gox.Attrs))
	for i, attr := range gox.Attrs {
		if attr.Rhs == nil { // default to true like JSX
			attr.Rhs = ast.NewIdent("true")
		}
		expr := &ast.KeyValueExpr{
			Key:   ast.NewIdent(attr.Lhs.Name),
			Colon: token.NoPos,
			Value: attr.Rhs,
		}

		args[i] = expr
	}

	if len(gox.X) != 0 {
		expr := &ast.KeyValueExpr{
			Key:   ast.NewIdent("Body"),
			Colon: token.NoPos,
			Value: newCallExpr(
				newSelectorExpr(genname, "Text"),
				append([]ast.Expr{
					&ast.BasicLit{
						ValuePos: token.NoPos,
						Value:    `""`,
						Kind:     token.STRING,
					},
				}, gox.X...),
			),
		}

		args = append(args, expr)
	}

	if t, ok := gox.TagName.(*ast.CallExpr); ok {
		if len(gox.X) != 0 {
			t.Args = append(t.Args, newCallExpr(
				newSelectorExpr(genname, "Text"),
				append([]ast.Expr{
					&ast.BasicLit{
						ValuePos: token.NoPos,
						Value:    `""`,
						Kind:     token.STRING,
					},
				}, gox.X...),
			))
		}
		return t
	}

	return newCallExpr(
		newSelectorExpr(genname, "NewComponent"),
		[]ast.Expr{
			&ast.UnaryExpr{
				OpPos: token.NoPos,
				Op:    token.AND,
				X: &ast.CompositeLit{
					Type:   gox.TagName,
					Lbrace: token.NoPos,
					Elts:   args,
					Rbrace: token.NoPos,
				},
			},
		},
	)
}

func mapProps(genname string, goxAttrs []*ast.GoxAttrStmt) []ast.Expr {
	var mapped = []ast.Expr{}
	for _, attr := range goxAttrs {
		// set default of Rhs to true if none provided
		if attr.Rhs == nil { // default to true like JSX
			attr.Rhs = ast.NewIdent("true")
		}

		var expr ast.Expr

		// if prop is an event listener (e.g. "onClick")
		if _, ok := eventMap[attr.Lhs.Name]; ok {
			expr = newEventListener(genname, attr)
		} else if mappedName, ok := attrMap[attr.Lhs.Name]; ok {
			// if it's a vecty controlled prop
			expr = newCallExpr(
				newSelectorExpr(genname, "Property"),
				[]ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(mappedName)},
					attr.Rhs,
				},
			)
		} else {
			// if prop is a normal attribute
			expr = newCallExpr(
				newSelectorExpr(genname, "Attribute"),
				[]ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(attr.Lhs.Name)},
					attr.Rhs,
				},
			)
		}

		mapped = append(mapped, expr)
	}

	return mapped
}

func newEventListener(genname string, goxAttr *ast.GoxAttrStmt) ast.Expr {
	return &ast.UnaryExpr{
		OpPos: token.NoPos,
		Op:    token.AND,
		X: &ast.CompositeLit{
			Type:   newSelectorExpr(genname, "EventListener"),
			Lbrace: token.NoPos,
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key: ast.NewIdent("Name"),
					Value: &ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(eventMap[goxAttr.Lhs.Name]),
					},
				},
				&ast.KeyValueExpr{
					Key:   ast.NewIdent("Listener"),
					Value: goxAttr.Rhs,
				},
			},
			Rbrace: token.NoPos,
		},
	}
}
