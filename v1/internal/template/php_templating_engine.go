package template

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/vrianta/agai/v1/utils"
	log "github.com/vrianta/gulog"
)

var phpBlockPattern = regexp.MustCompile(`<\?php([\s\S]*?)\?>`)

// / removing comments
var longComments = regexp.MustCompile(`/\*[\s\S]*?\*/`)
var oneLineComment = regexp.MustCompile(`//.*(\r?\n|$)`)

// var foreachPattern = regexp.MustCompile(`foreach\s*\(\s*(\${1,2}[a-zA-Z_][a-zA-Z0-9_]*(?:->\w+)*)\s+as\s+(\${1,2}[a-zA-Z_][a-zA-Z0-9_]*)(?:\s*=>\s*(\${1,2}[a-zA-Z_][a-zA-Z0-9_]*))?\s*\)\s*:?`)
// supports ->prop, ['key'], [0], and optional & before vars
// supports variables, chains, indexing, function calls, etc.
var foreachPattern = regexp.MustCompile(
	`foreach\s*\(\s*` +
		`(.+?)` + // collection expression (non-greedy)
		`\s+as\s+` +
		`(?:&\s*)?(\${1,2}[A-Za-z_]\w*)` + // key or value
		`(?:\s*=>\s*(?:&\s*)?(\${1,2}[A-Za-z_]\w*))?` + // optional value
		`\s*\)\s*:?`,
)

var endforeachPattern = regexp.MustCompile(`(?m)\bendforeach\b\s*[:;]?`)

var ifPattern = regexp.MustCompile(`if\s*\(((?:[^()]+|\((?:[^()]+|\([^()]*\))*\))*)\)\s*[:;]?`)

// var elseifPattern = regexp.MustCompile(`(?m)(?:else\s*if|elseif)\s*\(([\s\S]*?)\)\s*[:;]?`)
// var elseifPattern = regexp.MustCompile(`(?m)(?:else\s*if|elseif)\s*\(\s*([\s\S]*?)\s*\)\s*[:;]?`)
var elseifPattern = regexp.MustCompile(`(?m)(?:else\s*if|elseif)\s*\(((?:[^()]+|\((?:[^()]+|\([^()]*\))*\))*)\)\s*[:;]?`)

var elsePattern = regexp.MustCompile(`(?m)\belse\s*(?:[:;]|$)`)
var endifPattern = regexp.MustCompile(`(?m)\bendif\b\s*[:;]?`)

var forPattern = regexp.MustCompile(`(?m)for\s*\(\s*([^;]+)\s*;\s*([^;]+)\s*;\s*([^)]+)\)\s*:?`)

// Assignment pattern: $var = expr; or $$var = expr;
var assignmentPattern = regexp.MustCompile(`(\${1,2}[A-Za-z_]\w*)\s*=\s*([^;]+)(?:;)?`)

// PHPToGoTemplate converts PHP template code to Go html/template syntax
func PHPToGoTemplate(phpTemplate string) string {

	phpTemplate = phpBlockPattern.ReplaceAllStringFunc(phpTemplate, func(match string) string {
		// First extract inner code safely using the delimiters that are still present
		m := phpBlockPattern.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}

		// Now remove comments only inside the block
		inner := m[1]
		inner = oneLineComment.ReplaceAllString(inner, "")
		inner = longComments.ReplaceAllString(inner, "")
		code := strings.TrimSpace(inner)

		if code == "" {
			// Empty PHP block after stripping comments
			return ""
		}
		return ConvertPHPBlockToGo(code)
	})

	// // Step 1: Convert short echo tags <?= ... ?> to Go template syntax
	shortEchoPattern := regexp.MustCompile(`<\?=\s*(.*?)\s*\?>`)
	phpTemplate = shortEchoPattern.ReplaceAllStringFunc(phpTemplate, func(match string) string {
		// removing commnet
		match = oneLineComment.ReplaceAllString(match, "")
		// removing long commands /**/
		match = longComments.ReplaceAllString(match, "")

		expr := strings.TrimSpace(shortEchoPattern.FindStringSubmatch(match)[1])

		if expr == "" {
			// Empty PHP block after stripping comments
			return ""
		}
		return "{{ " + convertPHPExprToGo(expr) + " }}"
	})

	// fmt.Println(phpTemplate)
	return phpTemplate
}

func ConvertPHPBlockToGo(code string) string {

	code = foreachPattern.ReplaceAllStringFunc(code, func(match string) string {
		m := foreachPattern.FindStringSubmatch(match)
		if len(m) < 3 {
			return match
		}

		collection := convertPHPExprToGo(strings.TrimSpace(m[1]))

		clean := func(s string) string {
			s = strings.TrimSpace(s)
			s = strings.TrimPrefix(s, "&")
			return s
		}
		k := clean(m[2])
		v := clean(m[3])

		if v != "" {
			return fmt.Sprintf("{{ range %s, %s := %s }}", k, v, collection)
		}
		return fmt.Sprintf("{{ range %s := %s }}", k, collection)
	})

	// Replace all matches with Go template end
	code = endforeachPattern.ReplaceAllString(code, "{{ end }}")

	// Handle elseif / else if blocks in multi-line code
	code = elseifPattern.ReplaceAllStringFunc(code, func(match string) string {
		matches := elseifPattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match // fallback if regex fails
		}
		// condition := convertPHPExprToGo(matches[1])
		return fmt.Sprintf("{{ else if %s }}", convertPHPExprToGo(matches[1]))
	})

	code = ifPattern.ReplaceAllStringFunc(code, func(match string) string {
		matches := ifPattern.FindStringSubmatch(match)

		condition := convertPHPExprToGo(matches[1])
		return fmt.Sprintf("{{ if %s }}", condition)
	})

	// Handle else statements
	code = elsePattern.ReplaceAllString(code, "{{ else }}")

	// Handle endif statements

	code = endifPattern.ReplaceAllString(code, "{{ end }}")

	// Handle for loops (auto-convert common patterns)

	code = forPattern.ReplaceAllStringFunc(code, func(match string) string {
		m := forPattern.FindStringSubmatch(match)
		if len(m) < 4 {
			return match
		}
		init := strings.TrimSpace(m[1])
		cond := strings.TrimSpace(m[2])
		post := strings.TrimSpace(m[3])

		// case 1: for($i = 0; $i < count($arr); $i++)
		if im := regexp.MustCompile(`^\$([a-zA-Z_]\w*)\s*=\s*0$`).FindStringSubmatch(init); im != nil {
			idx := im[1]
			if cm := regexp.MustCompile(`^\$` + idx + `\s*<\s*count\s*\(\s*\$([a-zA-Z_]\w*)\s*\)\s*$`).FindStringSubmatch(cond); cm != nil {
				arr := cm[1]
				// $item name: use idx + "Item" to avoid collisions
				itemVar := idx + "Item"
				return fmt.Sprintf("{{ range $%s, $%s := .%s }}", idx, itemVar, arr)
			}
		}

		// case 2: for($i = 0; $i < $n; $i++) -> best-effort placeholder using seq helper
		if im := regexp.MustCompile(`^\$([a-zA-Z_]\w*)\s*=\s*0$`).FindStringSubmatch(init); im != nil {
			idx := im[1]
			if cm := regexp.MustCompile(`^\$` + idx + `\s*<\s*\$?([a-zA-Z_]\w*)\s*$`).FindStringSubmatch(cond); cm != nil {
				bound := cm[1]
				// Note: this assumes you provide a "seq" or equivalent template func that yields 0..bound-1
				// If you don't have seq, this is a documentation placeholder.
				return fmt.Sprintf("{{ /* FOR converted: for(%s; %s; %s) - verify and provide seq() if needed */ range $%s := seq 0 .%s }}", init, cond, post, idx, bound)
			}
		}

		// fallback: cannot auto-convert safely
		return fmt.Sprintf("{{ /* FOR loop not auto-converted: (%s; %s; %s) - review */ }}", init, cond, post)
	})

	// Handle endfor
	if code == "endfor" || code == "endfor;" || code == "endfor :" {
		return "{{ end }}"
	}

	// Handle while loops
	if strings.HasPrefix(code, "while") {
		// Go templates don't have direct while loop equivalents
		whilePattern := regexp.MustCompile(`while\s*\((.*?)\)(?:\s*:)?`)
		matches := whilePattern.FindStringSubmatch(code)
		if len(matches) > 1 {
			condition := convertPHPExprToGo(matches[1])
			return fmt.Sprintf("{{ /* WHILE loop conversion - REVIEW THIS */ }}\nif %s }}", condition)
		}
	}

	// Handle endwhile
	if code == "endwhile" || code == "endwhile;" || code == "endwhile :" {
		return "{{ end }}"
	}

	// Handle PHP echo statements
	if strings.HasPrefix(code, "echo") {
		echoPattern := regexp.MustCompile(`echo\s+(.*?)(;|\s*$)`)
		matches := echoPattern.FindStringSubmatch(code)
		if len(matches) > 1 {
			expr := convertPHPExprToGo(matches[1])
			return fmt.Sprintf("{{ %s }}", expr)
		}
	}

	// Handle variable assignments: $var = expr; (after control structures)
	// Only apply if the code doesn't contain already-converted template syntax
	if !strings.Contains(code, "{{") && !strings.Contains(code, "}}") {
		code = assignmentPattern.ReplaceAllStringFunc(code, func(match string) string {
			m := assignmentPattern.FindStringSubmatch(match)
			if len(m) < 3 {
				return match
			}

			varName := strings.TrimSpace(m[1])
			expr := strings.TrimSpace(m[2])

			// Convert the expression
			convertedExpr := convertPHPExprToGo(expr)

			// Convert variable name: $var becomes $var, $$var becomes $var
			if strings.HasPrefix(varName, "$$") {
				varName = "$" + varName[2:]
			}

			return fmt.Sprintf("{{ %s := %s }}", varName, convertedExpr)
		})
	}

	return code
}

func convertPHPExprToGo(expr string) string {
	expr = convertPHPVarsToGo(strings.TrimSpace(expr))

	functionCallPattern := regexp.MustCompile(`(\w+)\((.*?)\)`)
	// helper: normalize arg list to space-separated, handling nested parens
	normalizeArgs := func(argSrc string) string {
		argSrc = strings.TrimSpace(argSrc)
		if argSrc == "" {
			return ""
		}
		parts := splitTopLevel(argSrc, ',')
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			out = append(out, convertPHPVarsToGo(p))
		}
		return strings.Join(out, " ")
	}

	expr = functionCallPattern.ReplaceAllStringFunc(expr, func(match string) string {
		parts := functionCallPattern.FindStringSubmatch(match)
		funcName := parts[1]
		args := normalizeArgs(parts[2])

		switch funcName {
		case "strtoupper":
			return fmt.Sprintf("Upper %s", args)
		case "strtolower":
			return fmt.Sprintf("lower %s", args)
		case "htmlspecialchars":
			return fmt.Sprintf("Html %s", args)
		case "isset":
			// isset(x) -> ne x nil
			return fmt.Sprintf("ne %s nil", args)
		case "empty":
			// empty(x) -> eq x ""
			return fmt.Sprintf("eq %s \"\"", args)
		case "print", "date", "len", "upper", "lower", "strlen", "count":
			return fmt.Sprintf("%s %s", funcName, args)
		case "array_values":
			return fmt.Sprintf("values %s", args)
		case "array_keys":
			return fmt.Sprintf("keys %s", args)
		case "include":
			_tInfo, ok := _templateInfo[args]
			if !ok {
				log.Error("Template not found: %s", args)
				return fmt.Sprintf("include %s .", args)
			}
			content := string(utils.ReadFromFile(_tInfo.folderPath))
			return fmt.Sprintf("include %s .", content)
		default:
			// unknown: call .foo arg1 arg2 ...
			if args != "" {
				// safety: collapse any top-level commas that slipped through
				cleaned := strings.Join(splitTopLevel(args, ','), " ")
				return fmt.Sprintf("call .%s %s", funcName, cleaned)
			}
			return fmt.Sprintf("call .%s", funcName)
		}
	})

	return expr
}

// converting php codes : you have to pass code inside <?php ?> it will convert them one by one

// convertPHPVarsToGo converts PHP-style variable expressions into Go template syntax.
//
// It handles the following transformations:
// - Object properties:         $$object->property        -> $object.property
// - Array (map) access:        $array['key']            -> array.key
// - Indexed arrays:            $array[0]                -> index array 0
// - Simple variables:          $var                     -> .var
// - Operators:
//     - ==                    -> eq
//     - !=                    -> ne
//     - <                     -> lt
//     - >                     -> gt
//     - <=                    -> le
//     - >=                    -> ge
//     - &&                    -> and
//     - ||                    -> or
//     - !                     -> not

// splitTopLevel splits s by sepRune only at top level (not inside parens or quotes).
func splitTopLevel(s string, sepRune rune) []string {
	var out []string
	start, depth := 0, 0
	quote := rune(0)
	esc := false
	for i, r := range s {
		if esc {
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == '(' {
			depth++
			continue
		}
		if r == ')' {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 && r == sepRune {
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

func convertLogical(expr string) string {
	expr = strings.TrimSpace(expr)

	// First convert inside parentheses without looping
	var out strings.Builder
	for i := 0; i < len(expr); {
		if expr[i] == '(' {
			// find matching ')'
			depth := 1
			j := i + 1
			for ; j < len(expr) && depth > 0; j++ {
				switch expr[j] {
				case '(':
					depth++
				case ')':
					depth--
				}
			}
			inner := convertLogical(expr[i+1 : j-1])
			out.WriteString("(")
			out.WriteString(inner)
			out.WriteString(")")
			i = j
		} else {
			out.WriteByte(expr[i])
			i++
		}
	}
	expr = out.String()

	// AND
	if strings.Contains(expr, "&&") {
		parts := splitTopLevel(expr, '&')
		var ops []string
		for i := 0; i < len(parts); {
			if i+1 < len(parts) && parts[i+1] == "" {
				ops = append(ops, strings.TrimSpace(parts[i]))
				i += 2
			} else {
				ops = append(ops, strings.TrimSpace(parts[i]))
				i++
			}
		}
		// REMOVE THIS:
		// for i := range ops { ops[i] = "(" + ops[i] + ")" }

		return "and " + strings.Join(ops, " ")
	}

	// OR
	if strings.Contains(expr, "||") {
		parts := splitTopLevel(expr, '|')
		var ops []string
		for i := 0; i < len(parts); {
			if i+1 < len(parts) && parts[i+1] == "" {
				ops = append(ops, strings.TrimSpace(parts[i]))
				i += 2
			} else {
				ops = append(ops, strings.TrimSpace(parts[i]))
				i++
			}
		}
		// REMOVE THIS:
		// for i := range ops { ops[i] = "(" + ops[i] + ")" }

		return "or " + strings.Join(ops, " ")
	}

	return expr
}

func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return s
	}

	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i < len(s)-1 {
				// closing paren before end means not a full wrapper
				return s
			}
		}
	}
	// if we reach here: outer parens wrap entire expr
	return stripOuterParens(s[1 : len(s)-1])
}

// convertPHPVarsToGo transforms a PHP-like expression into Go template-style prefix ops.
func convertPHPVarsToGo(expr string) string {
	expr = strings.TrimSpace(expr)

	// 0) COMPLEX ARRAY ACCESS: $$var[$complex->expr]->prop
	// Handles: $$event_themes[$enquiry->Aesthetic]->ThemeName
	// Only matches if there's a property access (->)  inside the brackets
	// Converts to: (index .event_themes .enquiry.Aesthetic).ThemeName
	complexArrChainPattern := regexp.MustCompile(`\${1,2}(\w+)\[(\$\w+(?:->[A-Za-z_]\w*)+)\](->[A-Za-z_]\w*)?`)
	expr = complexArrChainPattern.ReplaceAllStringFunc(expr, func(match string) string {
		m := complexArrChainPattern.FindStringSubmatch(match)
		varName := m[1]    // event_themes
		indexExpr := m[2]  // $enquiry->Aesthetic
		propAccess := m[3] // ->ThemeName or empty

		// Convert the index expression: $enquiry->Aesthetic becomes .enquiry.Aesthetic
		// First replace -> with .
		convertedIndex := strings.ReplaceAll(indexExpr, "->", ".")

		// Then convert leading $ or $$ to .
		// Remove leading $$ or $ and replace with .
		if strings.HasPrefix(convertedIndex, "$$") {
			convertedIndex = "." + convertedIndex[2:]
		} else if strings.HasPrefix(convertedIndex, "$") {
			convertedIndex = "." + convertedIndex[1:]
		}

		// Build the index expression: (index .varname .index_var)
		result := fmt.Sprintf("(index .%s %s)", varName, convertedIndex)

		// If there's property access after array, append it: .ThemeName
		if propAccess != "" {
			prop := strings.TrimPrefix(propAccess, "->")
			result = fmt.Sprintf("%s.%s", result, prop)
		}

		return result
	})

	// 1) Arrays: $a['k'] or $$a['k'] -> .a.k (also handles trailing ->property)
	reArrStr := regexp.MustCompile(`\${1,2}(\w+)\[\s*['"](\w+)['"]\s*\](?:->([\w]+))?`)
	expr = reArrStr.ReplaceAllStringFunc(expr, func(match string) string {
		m := reArrStr.FindStringSubmatch(match)
		varName := m[1] // themes
		key := m[2]     // default
		prop := m[3]    // Name or empty

		result := "." + varName + "." + key
		if prop != "" {
			result = result + "." + prop
		}
		return result
	})

	// Numeric: $a[0] or $$a[0] -> index .a 0
	reArrNum := regexp.MustCompile(`\${1,2}(\w+)\[\s*(\d+)\s*\]`)
	expr = reArrNum.ReplaceAllString(expr, `index .$1 $2`)

	// $a[$i] / $$a[$i] -> index $$a $i   (to be normalized later)
	reArrVar := regexp.MustCompile(`\${1,2}(\w+)\[\s*\$(\w+)\s*\]`)
	expr = reArrVar.ReplaceAllString(expr, `index $$$$$1 $$$2`)

	// Combined: $$obj->x, $obj->x, $$x, $x
	var varOrChain = regexp.MustCompile(`\${1,2}(?:[A-Za-z_]\w*(?:->\w+)+|[A-Za-z_]\w*)`)

	expr = varOrChain.ReplaceAllStringFunc(expr, func(s string) string {
		// detect $$ vs $
		double := len(s) >= 2 && s[1] == '$'

		// object chain?
		if strings.Contains(s, "->") {
			// Replace -> with .
			s = strings.ReplaceAll(s, "->", ".")
		}

		if double {
			return "." + s[2:] // $$var -> .var (access from context data)
		} else {
			return s // $var stays as $var (template variable)
		}
	})

	// 4) Binary comparisons to prefix ops
	binComp := regexp.MustCompile(`(\S+)\s*(==|!=|<=|>=|<|>)\s*(\S+)`)
	opMap := map[string]string{"==": "eq", "!=": "ne", "<=": "le", ">=": "ge", "<": "lt", ">": "gt"}
	expr = binComp.ReplaceAllStringFunc(expr, func(m string) string {
		p := binComp.FindStringSubmatch(m)
		return opMap[p[2]] + " " + strings.TrimSpace(p[1]) + " " + strings.TrimSpace(p[3])
	})

	// 5) Logical NOT
	reNot := regexp.MustCompile(`!\s*`)
	expr = reNot.ReplaceAllString(expr, `not `)

	wrap := func(x string) string {
		x = strings.TrimSpace(x)
		// If it's already a single token or already "(call ...)" do not wrap
		if strings.HasPrefix(x, "(") && strings.HasSuffix(x, ")") {
			return x
		}
		if strings.HasPrefix(x, ".") || regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(x) {
			return x // do NOT wrap simple identifiers
		}
		return "(" + x + ")"
	}

	// 6) Logical AND at top level -> prefix with parens
	if strings.Contains(expr, "&&") {
		parts := splitTopLevel(expr, '&') // splits on each '&'
		var ops []string
		for i := 0; i < len(parts); {
			if i+1 < len(parts) && parts[i+1] == "" { // "&&"
				ops = append(ops, strings.TrimSpace(parts[i]))
				i += 2
			} else {
				ops = append(ops, strings.TrimSpace(parts[i]))
				i++
			}
		}
		if len(ops) > 1 {
			for i := range ops {
				ops[i] = wrap(ops[i])
			}
			expr = "and " + strings.Join(ops, " ")
		} else if len(ops) == 1 {
			expr = ops[0]
		}
	}

	// 7) Logical OR at top level -> prefix with parens
	if strings.Contains(expr, "||") {
		parts := splitTopLevel(expr, '|')
		var ops []string
		for i := 0; i < len(parts); {
			if i+1 < len(parts) && parts[i+1] == "" { // "||"
				ops = append(ops, strings.TrimSpace(parts[i]))
				i += 2
			} else {
				ops = append(ops, strings.TrimSpace(parts[i]))
				i++
			}
		}
		if len(ops) > 1 {
			for i := range ops {
				ops[i] = wrap(ops[i])
			}
			expr = "or " + strings.Join(ops, " ")
		} else if len(ops) == 1 {
			expr = ops[0]
		}
	}
	expr = convertLogical(expr)
	expr = stripOuterParens(expr)

	return strings.TrimSpace(expr)
}
