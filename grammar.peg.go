package query

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleQuery
	ruleColumnExpr
	ruleGroupExpr
	ruleWhereExpr
	ruleOrderByExpr
	ruleLimitExpr
	ruleColumns
	ruleColumn
	ruleColumnAggregation
	ruleLogicExpr
	ruleOPERATOR
	ruleFilterKey
	ruleFilterOperator
	ruleFilterValue
	ruleDescending
	ruleString
	ruleStringChar
	ruleEscape
	ruleSimpleEscape
	ruleOctalEscape
	ruleHexEscape
	ruleUniversalCharacter
	ruleHexQuad
	ruleHexDigit
	ruleUnsigned
	ruleSign
	ruleInteger
	ruleFloat
	ruleIdentifier
	ruleIdChar
	ruleKeyword
	rule_
	ruleLPAR
	ruleRPAR
	ruleCOMMA
	ruleAction0
	ruleAction1
	ruleAction2
	rulePegText
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
)

var rul3s = [...]string{
	"Unknown",
	"Query",
	"ColumnExpr",
	"GroupExpr",
	"WhereExpr",
	"OrderByExpr",
	"LimitExpr",
	"Columns",
	"Column",
	"ColumnAggregation",
	"LogicExpr",
	"OPERATOR",
	"FilterKey",
	"FilterOperator",
	"FilterValue",
	"Descending",
	"String",
	"StringChar",
	"Escape",
	"SimpleEscape",
	"OctalEscape",
	"HexEscape",
	"UniversalCharacter",
	"HexQuad",
	"HexDigit",
	"Unsigned",
	"Sign",
	"Integer",
	"Float",
	"Identifier",
	"IdChar",
	"Keyword",
	"_",
	"LPAR",
	"RPAR",
	"COMMA",
	"Action0",
	"Action1",
	"Action2",
	"PegText",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type parser struct {
	expression

	Buffer string
	buffer []rune
	rules  [53]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *parser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *parser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *parser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *parser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *parser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.currentSection = "columns"
		case ruleAction1:
			p.currentSection = "group by"
		case ruleAction2:
			p.currentSection = "order by"
		case ruleAction3:
			p.SetLimit(text)
		case ruleAction4:
			p.AddColumn()
		case ruleAction5:
			p.SetColumnName(text)
		case ruleAction6:
			p.SetColumnName(text)
		case ruleAction7:
			p.SetColumnAggregate(text)
		case ruleAction8:
			p.SetColumnName(text)
		case ruleAction9:
			p.AddFilter()
		case ruleAction10:
			p.SetFilterColumn(text)
		case ruleAction11:
			p.SetFilterOperator(text)
		case ruleAction12:
			p.SetFilterValueFloat(text)
		case ruleAction13:
			p.SetFilterValueInteger(text)
		case ruleAction14:
			p.SetFilterValueString(text)
		case ruleAction15:
			p.SetDescending()

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *parser) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Query <- <(_ ColumnExpr? _ WhereExpr? _ GroupExpr? _ OrderByExpr? _ LimitExpr? _ !.)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[rule_]() {
					goto l0
				}
				{
					position2, tokenIndex2 := position, tokenIndex
					if !_rules[ruleColumnExpr]() {
						goto l2
					}
					goto l3
				l2:
					position, tokenIndex = position2, tokenIndex2
				}
			l3:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position4, tokenIndex4 := position, tokenIndex
					if !_rules[ruleWhereExpr]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex = position4, tokenIndex4
				}
			l5:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position6, tokenIndex6 := position, tokenIndex
					if !_rules[ruleGroupExpr]() {
						goto l6
					}
					goto l7
				l6:
					position, tokenIndex = position6, tokenIndex6
				}
			l7:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position8, tokenIndex8 := position, tokenIndex
					if !_rules[ruleOrderByExpr]() {
						goto l8
					}
					goto l9
				l8:
					position, tokenIndex = position8, tokenIndex8
				}
			l9:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position10, tokenIndex10 := position, tokenIndex
					if !_rules[ruleLimitExpr]() {
						goto l10
					}
					goto l11
				l10:
					position, tokenIndex = position10, tokenIndex10
				}
			l11:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position12, tokenIndex12 := position, tokenIndex
					if !matchDot() {
						goto l12
					}
					goto l0
				l12:
					position, tokenIndex = position12, tokenIndex12
				}
				add(ruleQuery, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 ColumnExpr <- <(('s' / 'S') ('e' / 'E') ('l' / 'L') ('e' / 'E') ('c' / 'C') ('t' / 'T') _ Action0 Columns)> */
		func() bool {
			position13, tokenIndex13 := position, tokenIndex
			{
				position14 := position
				{
					position15, tokenIndex15 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l16
					}
					position++
					goto l15
				l16:
					position, tokenIndex = position15, tokenIndex15
					if buffer[position] != rune('S') {
						goto l13
					}
					position++
				}
			l15:
				{
					position17, tokenIndex17 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l18
					}
					position++
					goto l17
				l18:
					position, tokenIndex = position17, tokenIndex17
					if buffer[position] != rune('E') {
						goto l13
					}
					position++
				}
			l17:
				{
					position19, tokenIndex19 := position, tokenIndex
					if buffer[position] != rune('l') {
						goto l20
					}
					position++
					goto l19
				l20:
					position, tokenIndex = position19, tokenIndex19
					if buffer[position] != rune('L') {
						goto l13
					}
					position++
				}
			l19:
				{
					position21, tokenIndex21 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l22
					}
					position++
					goto l21
				l22:
					position, tokenIndex = position21, tokenIndex21
					if buffer[position] != rune('E') {
						goto l13
					}
					position++
				}
			l21:
				{
					position23, tokenIndex23 := position, tokenIndex
					if buffer[position] != rune('c') {
						goto l24
					}
					position++
					goto l23
				l24:
					position, tokenIndex = position23, tokenIndex23
					if buffer[position] != rune('C') {
						goto l13
					}
					position++
				}
			l23:
				{
					position25, tokenIndex25 := position, tokenIndex
					if buffer[position] != rune('t') {
						goto l26
					}
					position++
					goto l25
				l26:
					position, tokenIndex = position25, tokenIndex25
					if buffer[position] != rune('T') {
						goto l13
					}
					position++
				}
			l25:
				if !_rules[rule_]() {
					goto l13
				}
				if !_rules[ruleAction0]() {
					goto l13
				}
				if !_rules[ruleColumns]() {
					goto l13
				}
				add(ruleColumnExpr, position14)
			}
			return true
		l13:
			position, tokenIndex = position13, tokenIndex13
			return false
		},
		/* 2 GroupExpr <- <(('g' / 'G') ('r' / 'R') ('o' / 'O') ('u' / 'U') ('p' / 'P') ' ' ('b' / 'B') ('y' / 'Y') _ Action1 Columns)> */
		func() bool {
			position27, tokenIndex27 := position, tokenIndex
			{
				position28 := position
				{
					position29, tokenIndex29 := position, tokenIndex
					if buffer[position] != rune('g') {
						goto l30
					}
					position++
					goto l29
				l30:
					position, tokenIndex = position29, tokenIndex29
					if buffer[position] != rune('G') {
						goto l27
					}
					position++
				}
			l29:
				{
					position31, tokenIndex31 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l32
					}
					position++
					goto l31
				l32:
					position, tokenIndex = position31, tokenIndex31
					if buffer[position] != rune('R') {
						goto l27
					}
					position++
				}
			l31:
				{
					position33, tokenIndex33 := position, tokenIndex
					if buffer[position] != rune('o') {
						goto l34
					}
					position++
					goto l33
				l34:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('O') {
						goto l27
					}
					position++
				}
			l33:
				{
					position35, tokenIndex35 := position, tokenIndex
					if buffer[position] != rune('u') {
						goto l36
					}
					position++
					goto l35
				l36:
					position, tokenIndex = position35, tokenIndex35
					if buffer[position] != rune('U') {
						goto l27
					}
					position++
				}
			l35:
				{
					position37, tokenIndex37 := position, tokenIndex
					if buffer[position] != rune('p') {
						goto l38
					}
					position++
					goto l37
				l38:
					position, tokenIndex = position37, tokenIndex37
					if buffer[position] != rune('P') {
						goto l27
					}
					position++
				}
			l37:
				if buffer[position] != rune(' ') {
					goto l27
				}
				position++
				{
					position39, tokenIndex39 := position, tokenIndex
					if buffer[position] != rune('b') {
						goto l40
					}
					position++
					goto l39
				l40:
					position, tokenIndex = position39, tokenIndex39
					if buffer[position] != rune('B') {
						goto l27
					}
					position++
				}
			l39:
				{
					position41, tokenIndex41 := position, tokenIndex
					if buffer[position] != rune('y') {
						goto l42
					}
					position++
					goto l41
				l42:
					position, tokenIndex = position41, tokenIndex41
					if buffer[position] != rune('Y') {
						goto l27
					}
					position++
				}
			l41:
				if !_rules[rule_]() {
					goto l27
				}
				if !_rules[ruleAction1]() {
					goto l27
				}
				if !_rules[ruleColumns]() {
					goto l27
				}
				add(ruleGroupExpr, position28)
			}
			return true
		l27:
			position, tokenIndex = position27, tokenIndex27
			return false
		},
		/* 3 WhereExpr <- <(('w' / 'W') ('h' / 'H') ('e' / 'E') ('r' / 'R') ('e' / 'E') _ LogicExpr (_ COMMA? LogicExpr)*)> */
		func() bool {
			position43, tokenIndex43 := position, tokenIndex
			{
				position44 := position
				{
					position45, tokenIndex45 := position, tokenIndex
					if buffer[position] != rune('w') {
						goto l46
					}
					position++
					goto l45
				l46:
					position, tokenIndex = position45, tokenIndex45
					if buffer[position] != rune('W') {
						goto l43
					}
					position++
				}
			l45:
				{
					position47, tokenIndex47 := position, tokenIndex
					if buffer[position] != rune('h') {
						goto l48
					}
					position++
					goto l47
				l48:
					position, tokenIndex = position47, tokenIndex47
					if buffer[position] != rune('H') {
						goto l43
					}
					position++
				}
			l47:
				{
					position49, tokenIndex49 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l50
					}
					position++
					goto l49
				l50:
					position, tokenIndex = position49, tokenIndex49
					if buffer[position] != rune('E') {
						goto l43
					}
					position++
				}
			l49:
				{
					position51, tokenIndex51 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l52
					}
					position++
					goto l51
				l52:
					position, tokenIndex = position51, tokenIndex51
					if buffer[position] != rune('R') {
						goto l43
					}
					position++
				}
			l51:
				{
					position53, tokenIndex53 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l54
					}
					position++
					goto l53
				l54:
					position, tokenIndex = position53, tokenIndex53
					if buffer[position] != rune('E') {
						goto l43
					}
					position++
				}
			l53:
				if !_rules[rule_]() {
					goto l43
				}
				if !_rules[ruleLogicExpr]() {
					goto l43
				}
			l55:
				{
					position56, tokenIndex56 := position, tokenIndex
					if !_rules[rule_]() {
						goto l56
					}
					{
						position57, tokenIndex57 := position, tokenIndex
						if !_rules[ruleCOMMA]() {
							goto l57
						}
						goto l58
					l57:
						position, tokenIndex = position57, tokenIndex57
					}
				l58:
					if !_rules[ruleLogicExpr]() {
						goto l56
					}
					goto l55
				l56:
					position, tokenIndex = position56, tokenIndex56
				}
				add(ruleWhereExpr, position44)
			}
			return true
		l43:
			position, tokenIndex = position43, tokenIndex43
			return false
		},
		/* 4 OrderByExpr <- <(('o' / 'O') ('r' / 'R') ('d' / 'D') ('e' / 'E') ('r' / 'R') ' ' ('b' / 'B') ('y' / 'Y') _ Action2 Columns Descending?)> */
		func() bool {
			position59, tokenIndex59 := position, tokenIndex
			{
				position60 := position
				{
					position61, tokenIndex61 := position, tokenIndex
					if buffer[position] != rune('o') {
						goto l62
					}
					position++
					goto l61
				l62:
					position, tokenIndex = position61, tokenIndex61
					if buffer[position] != rune('O') {
						goto l59
					}
					position++
				}
			l61:
				{
					position63, tokenIndex63 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l64
					}
					position++
					goto l63
				l64:
					position, tokenIndex = position63, tokenIndex63
					if buffer[position] != rune('R') {
						goto l59
					}
					position++
				}
			l63:
				{
					position65, tokenIndex65 := position, tokenIndex
					if buffer[position] != rune('d') {
						goto l66
					}
					position++
					goto l65
				l66:
					position, tokenIndex = position65, tokenIndex65
					if buffer[position] != rune('D') {
						goto l59
					}
					position++
				}
			l65:
				{
					position67, tokenIndex67 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l68
					}
					position++
					goto l67
				l68:
					position, tokenIndex = position67, tokenIndex67
					if buffer[position] != rune('E') {
						goto l59
					}
					position++
				}
			l67:
				{
					position69, tokenIndex69 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l70
					}
					position++
					goto l69
				l70:
					position, tokenIndex = position69, tokenIndex69
					if buffer[position] != rune('R') {
						goto l59
					}
					position++
				}
			l69:
				if buffer[position] != rune(' ') {
					goto l59
				}
				position++
				{
					position71, tokenIndex71 := position, tokenIndex
					if buffer[position] != rune('b') {
						goto l72
					}
					position++
					goto l71
				l72:
					position, tokenIndex = position71, tokenIndex71
					if buffer[position] != rune('B') {
						goto l59
					}
					position++
				}
			l71:
				{
					position73, tokenIndex73 := position, tokenIndex
					if buffer[position] != rune('y') {
						goto l74
					}
					position++
					goto l73
				l74:
					position, tokenIndex = position73, tokenIndex73
					if buffer[position] != rune('Y') {
						goto l59
					}
					position++
				}
			l73:
				if !_rules[rule_]() {
					goto l59
				}
				if !_rules[ruleAction2]() {
					goto l59
				}
				if !_rules[ruleColumns]() {
					goto l59
				}
				{
					position75, tokenIndex75 := position, tokenIndex
					if !_rules[ruleDescending]() {
						goto l75
					}
					goto l76
				l75:
					position, tokenIndex = position75, tokenIndex75
				}
			l76:
				add(ruleOrderByExpr, position60)
			}
			return true
		l59:
			position, tokenIndex = position59, tokenIndex59
			return false
		},
		/* 5 LimitExpr <- <(('l' / 'L') ('i' / 'I') ('m' / 'M') ('i' / 'I') ('t' / 'T') _ <Unsigned> Action3)> */
		func() bool {
			position77, tokenIndex77 := position, tokenIndex
			{
				position78 := position
				{
					position79, tokenIndex79 := position, tokenIndex
					if buffer[position] != rune('l') {
						goto l80
					}
					position++
					goto l79
				l80:
					position, tokenIndex = position79, tokenIndex79
					if buffer[position] != rune('L') {
						goto l77
					}
					position++
				}
			l79:
				{
					position81, tokenIndex81 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l82
					}
					position++
					goto l81
				l82:
					position, tokenIndex = position81, tokenIndex81
					if buffer[position] != rune('I') {
						goto l77
					}
					position++
				}
			l81:
				{
					position83, tokenIndex83 := position, tokenIndex
					if buffer[position] != rune('m') {
						goto l84
					}
					position++
					goto l83
				l84:
					position, tokenIndex = position83, tokenIndex83
					if buffer[position] != rune('M') {
						goto l77
					}
					position++
				}
			l83:
				{
					position85, tokenIndex85 := position, tokenIndex
					if buffer[position] != rune('i') {
						goto l86
					}
					position++
					goto l85
				l86:
					position, tokenIndex = position85, tokenIndex85
					if buffer[position] != rune('I') {
						goto l77
					}
					position++
				}
			l85:
				{
					position87, tokenIndex87 := position, tokenIndex
					if buffer[position] != rune('t') {
						goto l88
					}
					position++
					goto l87
				l88:
					position, tokenIndex = position87, tokenIndex87
					if buffer[position] != rune('T') {
						goto l77
					}
					position++
				}
			l87:
				if !_rules[rule_]() {
					goto l77
				}
				{
					position89 := position
					if !_rules[ruleUnsigned]() {
						goto l77
					}
					add(rulePegText, position89)
				}
				if !_rules[ruleAction3]() {
					goto l77
				}
				add(ruleLimitExpr, position78)
			}
			return true
		l77:
			position, tokenIndex = position77, tokenIndex77
			return false
		},
		/* 6 Columns <- <(Column (COMMA Column)*)> */
		func() bool {
			position90, tokenIndex90 := position, tokenIndex
			{
				position91 := position
				if !_rules[ruleColumn]() {
					goto l90
				}
			l92:
				{
					position93, tokenIndex93 := position, tokenIndex
					if !_rules[ruleCOMMA]() {
						goto l93
					}
					if !_rules[ruleColumn]() {
						goto l93
					}
					goto l92
				l93:
					position, tokenIndex = position93, tokenIndex93
				}
				add(ruleColumns, position91)
			}
			return true
		l90:
			position, tokenIndex = position90, tokenIndex90
			return false
		},
		/* 7 Column <- <(Action4 (ColumnAggregation / (<Identifier> _ Action5) / (<'*'> _ Action6)))> */
		func() bool {
			position94, tokenIndex94 := position, tokenIndex
			{
				position95 := position
				if !_rules[ruleAction4]() {
					goto l94
				}
				{
					position96, tokenIndex96 := position, tokenIndex
					if !_rules[ruleColumnAggregation]() {
						goto l97
					}
					goto l96
				l97:
					position, tokenIndex = position96, tokenIndex96
					{
						position99 := position
						if !_rules[ruleIdentifier]() {
							goto l98
						}
						add(rulePegText, position99)
					}
					if !_rules[rule_]() {
						goto l98
					}
					if !_rules[ruleAction5]() {
						goto l98
					}
					goto l96
				l98:
					position, tokenIndex = position96, tokenIndex96
					{
						position100 := position
						if buffer[position] != rune('*') {
							goto l94
						}
						position++
						add(rulePegText, position100)
					}
					if !_rules[rule_]() {
						goto l94
					}
					if !_rules[ruleAction6]() {
						goto l94
					}
				}
			l96:
				add(ruleColumn, position95)
			}
			return true
		l94:
			position, tokenIndex = position94, tokenIndex94
			return false
		},
		/* 8 ColumnAggregation <- <(<Identifier> Action7 LPAR <Identifier> RPAR Action8)> */
		func() bool {
			position101, tokenIndex101 := position, tokenIndex
			{
				position102 := position
				{
					position103 := position
					if !_rules[ruleIdentifier]() {
						goto l101
					}
					add(rulePegText, position103)
				}
				if !_rules[ruleAction7]() {
					goto l101
				}
				if !_rules[ruleLPAR]() {
					goto l101
				}
				{
					position104 := position
					if !_rules[ruleIdentifier]() {
						goto l101
					}
					add(rulePegText, position104)
				}
				if !_rules[ruleRPAR]() {
					goto l101
				}
				if !_rules[ruleAction8]() {
					goto l101
				}
				add(ruleColumnAggregation, position102)
			}
			return true
		l101:
			position, tokenIndex = position101, tokenIndex101
			return false
		},
		/* 9 LogicExpr <- <((LPAR LogicExpr RPAR) / (Action9 FilterKey _ FilterOperator _ FilterValue))> */
		func() bool {
			position105, tokenIndex105 := position, tokenIndex
			{
				position106 := position
				{
					position107, tokenIndex107 := position, tokenIndex
					if !_rules[ruleLPAR]() {
						goto l108
					}
					if !_rules[ruleLogicExpr]() {
						goto l108
					}
					if !_rules[ruleRPAR]() {
						goto l108
					}
					goto l107
				l108:
					position, tokenIndex = position107, tokenIndex107
					if !_rules[ruleAction9]() {
						goto l105
					}
					if !_rules[ruleFilterKey]() {
						goto l105
					}
					if !_rules[rule_]() {
						goto l105
					}
					if !_rules[ruleFilterOperator]() {
						goto l105
					}
					if !_rules[rule_]() {
						goto l105
					}
					if !_rules[ruleFilterValue]() {
						goto l105
					}
				}
			l107:
				add(ruleLogicExpr, position106)
			}
			return true
		l105:
			position, tokenIndex = position105, tokenIndex105
			return false
		},
		/* 10 OPERATOR <- <('=' / ('!' '=') / ('<' '=') / ('>' '=') / '<' / '>' / (('m' / 'M') ('a' / 'A') ('t' / 'T') ('c' / 'C') ('h' / 'H') ('e' / 'E') ('s' / 'S')))> */
		func() bool {
			position109, tokenIndex109 := position, tokenIndex
			{
				position110 := position
				{
					position111, tokenIndex111 := position, tokenIndex
					if buffer[position] != rune('=') {
						goto l112
					}
					position++
					goto l111
				l112:
					position, tokenIndex = position111, tokenIndex111
					if buffer[position] != rune('!') {
						goto l113
					}
					position++
					if buffer[position] != rune('=') {
						goto l113
					}
					position++
					goto l111
				l113:
					position, tokenIndex = position111, tokenIndex111
					if buffer[position] != rune('<') {
						goto l114
					}
					position++
					if buffer[position] != rune('=') {
						goto l114
					}
					position++
					goto l111
				l114:
					position, tokenIndex = position111, tokenIndex111
					if buffer[position] != rune('>') {
						goto l115
					}
					position++
					if buffer[position] != rune('=') {
						goto l115
					}
					position++
					goto l111
				l115:
					position, tokenIndex = position111, tokenIndex111
					if buffer[position] != rune('<') {
						goto l116
					}
					position++
					goto l111
				l116:
					position, tokenIndex = position111, tokenIndex111
					if buffer[position] != rune('>') {
						goto l117
					}
					position++
					goto l111
				l117:
					position, tokenIndex = position111, tokenIndex111
					{
						position118, tokenIndex118 := position, tokenIndex
						if buffer[position] != rune('m') {
							goto l119
						}
						position++
						goto l118
					l119:
						position, tokenIndex = position118, tokenIndex118
						if buffer[position] != rune('M') {
							goto l109
						}
						position++
					}
				l118:
					{
						position120, tokenIndex120 := position, tokenIndex
						if buffer[position] != rune('a') {
							goto l121
						}
						position++
						goto l120
					l121:
						position, tokenIndex = position120, tokenIndex120
						if buffer[position] != rune('A') {
							goto l109
						}
						position++
					}
				l120:
					{
						position122, tokenIndex122 := position, tokenIndex
						if buffer[position] != rune('t') {
							goto l123
						}
						position++
						goto l122
					l123:
						position, tokenIndex = position122, tokenIndex122
						if buffer[position] != rune('T') {
							goto l109
						}
						position++
					}
				l122:
					{
						position124, tokenIndex124 := position, tokenIndex
						if buffer[position] != rune('c') {
							goto l125
						}
						position++
						goto l124
					l125:
						position, tokenIndex = position124, tokenIndex124
						if buffer[position] != rune('C') {
							goto l109
						}
						position++
					}
				l124:
					{
						position126, tokenIndex126 := position, tokenIndex
						if buffer[position] != rune('h') {
							goto l127
						}
						position++
						goto l126
					l127:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune('H') {
							goto l109
						}
						position++
					}
				l126:
					{
						position128, tokenIndex128 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l129
						}
						position++
						goto l128
					l129:
						position, tokenIndex = position128, tokenIndex128
						if buffer[position] != rune('E') {
							goto l109
						}
						position++
					}
				l128:
					{
						position130, tokenIndex130 := position, tokenIndex
						if buffer[position] != rune('s') {
							goto l131
						}
						position++
						goto l130
					l131:
						position, tokenIndex = position130, tokenIndex130
						if buffer[position] != rune('S') {
							goto l109
						}
						position++
					}
				l130:
				}
			l111:
				add(ruleOPERATOR, position110)
			}
			return true
		l109:
			position, tokenIndex = position109, tokenIndex109
			return false
		},
		/* 11 FilterKey <- <(<Identifier> Action10)> */
		func() bool {
			position132, tokenIndex132 := position, tokenIndex
			{
				position133 := position
				{
					position134 := position
					if !_rules[ruleIdentifier]() {
						goto l132
					}
					add(rulePegText, position134)
				}
				if !_rules[ruleAction10]() {
					goto l132
				}
				add(ruleFilterKey, position133)
			}
			return true
		l132:
			position, tokenIndex = position132, tokenIndex132
			return false
		},
		/* 12 FilterOperator <- <(<OPERATOR> Action11)> */
		func() bool {
			position135, tokenIndex135 := position, tokenIndex
			{
				position136 := position
				{
					position137 := position
					if !_rules[ruleOPERATOR]() {
						goto l135
					}
					add(rulePegText, position137)
				}
				if !_rules[ruleAction11]() {
					goto l135
				}
				add(ruleFilterOperator, position136)
			}
			return true
		l135:
			position, tokenIndex = position135, tokenIndex135
			return false
		},
		/* 13 FilterValue <- <((<Float> Action12) / (<Integer> Action13) / (<String> Action14))> */
		func() bool {
			position138, tokenIndex138 := position, tokenIndex
			{
				position139 := position
				{
					position140, tokenIndex140 := position, tokenIndex
					{
						position142 := position
						if !_rules[ruleFloat]() {
							goto l141
						}
						add(rulePegText, position142)
					}
					if !_rules[ruleAction12]() {
						goto l141
					}
					goto l140
				l141:
					position, tokenIndex = position140, tokenIndex140
					{
						position144 := position
						if !_rules[ruleInteger]() {
							goto l143
						}
						add(rulePegText, position144)
					}
					if !_rules[ruleAction13]() {
						goto l143
					}
					goto l140
				l143:
					position, tokenIndex = position140, tokenIndex140
					{
						position145 := position
						if !_rules[ruleString]() {
							goto l138
						}
						add(rulePegText, position145)
					}
					if !_rules[ruleAction14]() {
						goto l138
					}
				}
			l140:
				add(ruleFilterValue, position139)
			}
			return true
		l138:
			position, tokenIndex = position138, tokenIndex138
			return false
		},
		/* 14 Descending <- <(('d' / 'D') ('e' / 'E') ('s' / 'S') ('c' / 'C') Action15)> */
		func() bool {
			position146, tokenIndex146 := position, tokenIndex
			{
				position147 := position
				{
					position148, tokenIndex148 := position, tokenIndex
					if buffer[position] != rune('d') {
						goto l149
					}
					position++
					goto l148
				l149:
					position, tokenIndex = position148, tokenIndex148
					if buffer[position] != rune('D') {
						goto l146
					}
					position++
				}
			l148:
				{
					position150, tokenIndex150 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l151
					}
					position++
					goto l150
				l151:
					position, tokenIndex = position150, tokenIndex150
					if buffer[position] != rune('E') {
						goto l146
					}
					position++
				}
			l150:
				{
					position152, tokenIndex152 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l153
					}
					position++
					goto l152
				l153:
					position, tokenIndex = position152, tokenIndex152
					if buffer[position] != rune('S') {
						goto l146
					}
					position++
				}
			l152:
				{
					position154, tokenIndex154 := position, tokenIndex
					if buffer[position] != rune('c') {
						goto l155
					}
					position++
					goto l154
				l155:
					position, tokenIndex = position154, tokenIndex154
					if buffer[position] != rune('C') {
						goto l146
					}
					position++
				}
			l154:
				if !_rules[ruleAction15]() {
					goto l146
				}
				add(ruleDescending, position147)
			}
			return true
		l146:
			position, tokenIndex = position146, tokenIndex146
			return false
		},
		/* 15 String <- <('"' <StringChar*> '"')+> */
		func() bool {
			position156, tokenIndex156 := position, tokenIndex
			{
				position157 := position
				if buffer[position] != rune('"') {
					goto l156
				}
				position++
				{
					position160 := position
				l161:
					{
						position162, tokenIndex162 := position, tokenIndex
						if !_rules[ruleStringChar]() {
							goto l162
						}
						goto l161
					l162:
						position, tokenIndex = position162, tokenIndex162
					}
					add(rulePegText, position160)
				}
				if buffer[position] != rune('"') {
					goto l156
				}
				position++
			l158:
				{
					position159, tokenIndex159 := position, tokenIndex
					if buffer[position] != rune('"') {
						goto l159
					}
					position++
					{
						position163 := position
					l164:
						{
							position165, tokenIndex165 := position, tokenIndex
							if !_rules[ruleStringChar]() {
								goto l165
							}
							goto l164
						l165:
							position, tokenIndex = position165, tokenIndex165
						}
						add(rulePegText, position163)
					}
					if buffer[position] != rune('"') {
						goto l159
					}
					position++
					goto l158
				l159:
					position, tokenIndex = position159, tokenIndex159
				}
				add(ruleString, position157)
			}
			return true
		l156:
			position, tokenIndex = position156, tokenIndex156
			return false
		},
		/* 16 StringChar <- <(Escape / (!('"' / '\n' / '\\') .))> */
		func() bool {
			position166, tokenIndex166 := position, tokenIndex
			{
				position167 := position
				{
					position168, tokenIndex168 := position, tokenIndex
					if !_rules[ruleEscape]() {
						goto l169
					}
					goto l168
				l169:
					position, tokenIndex = position168, tokenIndex168
					{
						position170, tokenIndex170 := position, tokenIndex
						{
							position171, tokenIndex171 := position, tokenIndex
							if buffer[position] != rune('"') {
								goto l172
							}
							position++
							goto l171
						l172:
							position, tokenIndex = position171, tokenIndex171
							if buffer[position] != rune('\n') {
								goto l173
							}
							position++
							goto l171
						l173:
							position, tokenIndex = position171, tokenIndex171
							if buffer[position] != rune('\\') {
								goto l170
							}
							position++
						}
					l171:
						goto l166
					l170:
						position, tokenIndex = position170, tokenIndex170
					}
					if !matchDot() {
						goto l166
					}
				}
			l168:
				add(ruleStringChar, position167)
			}
			return true
		l166:
			position, tokenIndex = position166, tokenIndex166
			return false
		},
		/* 17 Escape <- <(SimpleEscape / OctalEscape / HexEscape / UniversalCharacter)> */
		func() bool {
			position174, tokenIndex174 := position, tokenIndex
			{
				position175 := position
				{
					position176, tokenIndex176 := position, tokenIndex
					if !_rules[ruleSimpleEscape]() {
						goto l177
					}
					goto l176
				l177:
					position, tokenIndex = position176, tokenIndex176
					if !_rules[ruleOctalEscape]() {
						goto l178
					}
					goto l176
				l178:
					position, tokenIndex = position176, tokenIndex176
					if !_rules[ruleHexEscape]() {
						goto l179
					}
					goto l176
				l179:
					position, tokenIndex = position176, tokenIndex176
					if !_rules[ruleUniversalCharacter]() {
						goto l174
					}
				}
			l176:
				add(ruleEscape, position175)
			}
			return true
		l174:
			position, tokenIndex = position174, tokenIndex174
			return false
		},
		/* 18 SimpleEscape <- <('\\' ('\'' / '"' / '?' / '\\' / 'a' / 'b' / 'f' / 'n' / 'r' / 't' / 'v'))> */
		func() bool {
			position180, tokenIndex180 := position, tokenIndex
			{
				position181 := position
				if buffer[position] != rune('\\') {
					goto l180
				}
				position++
				{
					position182, tokenIndex182 := position, tokenIndex
					if buffer[position] != rune('\'') {
						goto l183
					}
					position++
					goto l182
				l183:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('"') {
						goto l184
					}
					position++
					goto l182
				l184:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('?') {
						goto l185
					}
					position++
					goto l182
				l185:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('\\') {
						goto l186
					}
					position++
					goto l182
				l186:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('a') {
						goto l187
					}
					position++
					goto l182
				l187:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('b') {
						goto l188
					}
					position++
					goto l182
				l188:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('f') {
						goto l189
					}
					position++
					goto l182
				l189:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('n') {
						goto l190
					}
					position++
					goto l182
				l190:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('r') {
						goto l191
					}
					position++
					goto l182
				l191:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('t') {
						goto l192
					}
					position++
					goto l182
				l192:
					position, tokenIndex = position182, tokenIndex182
					if buffer[position] != rune('v') {
						goto l180
					}
					position++
				}
			l182:
				add(ruleSimpleEscape, position181)
			}
			return true
		l180:
			position, tokenIndex = position180, tokenIndex180
			return false
		},
		/* 19 OctalEscape <- <('\\' [0-7] [0-7]? [0-7]?)> */
		func() bool {
			position193, tokenIndex193 := position, tokenIndex
			{
				position194 := position
				if buffer[position] != rune('\\') {
					goto l193
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('7') {
					goto l193
				}
				position++
				{
					position195, tokenIndex195 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('7') {
						goto l195
					}
					position++
					goto l196
				l195:
					position, tokenIndex = position195, tokenIndex195
				}
			l196:
				{
					position197, tokenIndex197 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('7') {
						goto l197
					}
					position++
					goto l198
				l197:
					position, tokenIndex = position197, tokenIndex197
				}
			l198:
				add(ruleOctalEscape, position194)
			}
			return true
		l193:
			position, tokenIndex = position193, tokenIndex193
			return false
		},
		/* 20 HexEscape <- <('\\' 'x' HexDigit+)> */
		func() bool {
			position199, tokenIndex199 := position, tokenIndex
			{
				position200 := position
				if buffer[position] != rune('\\') {
					goto l199
				}
				position++
				if buffer[position] != rune('x') {
					goto l199
				}
				position++
				if !_rules[ruleHexDigit]() {
					goto l199
				}
			l201:
				{
					position202, tokenIndex202 := position, tokenIndex
					if !_rules[ruleHexDigit]() {
						goto l202
					}
					goto l201
				l202:
					position, tokenIndex = position202, tokenIndex202
				}
				add(ruleHexEscape, position200)
			}
			return true
		l199:
			position, tokenIndex = position199, tokenIndex199
			return false
		},
		/* 21 UniversalCharacter <- <(('\\' 'u' HexQuad) / ('\\' 'U' HexQuad HexQuad))> */
		func() bool {
			position203, tokenIndex203 := position, tokenIndex
			{
				position204 := position
				{
					position205, tokenIndex205 := position, tokenIndex
					if buffer[position] != rune('\\') {
						goto l206
					}
					position++
					if buffer[position] != rune('u') {
						goto l206
					}
					position++
					if !_rules[ruleHexQuad]() {
						goto l206
					}
					goto l205
				l206:
					position, tokenIndex = position205, tokenIndex205
					if buffer[position] != rune('\\') {
						goto l203
					}
					position++
					if buffer[position] != rune('U') {
						goto l203
					}
					position++
					if !_rules[ruleHexQuad]() {
						goto l203
					}
					if !_rules[ruleHexQuad]() {
						goto l203
					}
				}
			l205:
				add(ruleUniversalCharacter, position204)
			}
			return true
		l203:
			position, tokenIndex = position203, tokenIndex203
			return false
		},
		/* 22 HexQuad <- <(HexDigit HexDigit HexDigit HexDigit)> */
		func() bool {
			position207, tokenIndex207 := position, tokenIndex
			{
				position208 := position
				if !_rules[ruleHexDigit]() {
					goto l207
				}
				if !_rules[ruleHexDigit]() {
					goto l207
				}
				if !_rules[ruleHexDigit]() {
					goto l207
				}
				if !_rules[ruleHexDigit]() {
					goto l207
				}
				add(ruleHexQuad, position208)
			}
			return true
		l207:
			position, tokenIndex = position207, tokenIndex207
			return false
		},
		/* 23 HexDigit <- <([a-f] / [A-F] / [0-9])> */
		func() bool {
			position209, tokenIndex209 := position, tokenIndex
			{
				position210 := position
				{
					position211, tokenIndex211 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('f') {
						goto l212
					}
					position++
					goto l211
				l212:
					position, tokenIndex = position211, tokenIndex211
					if c := buffer[position]; c < rune('A') || c > rune('F') {
						goto l213
					}
					position++
					goto l211
				l213:
					position, tokenIndex = position211, tokenIndex211
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l209
					}
					position++
				}
			l211:
				add(ruleHexDigit, position210)
			}
			return true
		l209:
			position, tokenIndex = position209, tokenIndex209
			return false
		},
		/* 24 Unsigned <- <[0-9]+> */
		func() bool {
			position214, tokenIndex214 := position, tokenIndex
			{
				position215 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l214
				}
				position++
			l216:
				{
					position217, tokenIndex217 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l217
					}
					position++
					goto l216
				l217:
					position, tokenIndex = position217, tokenIndex217
				}
				add(ruleUnsigned, position215)
			}
			return true
		l214:
			position, tokenIndex = position214, tokenIndex214
			return false
		},
		/* 25 Sign <- <('-' / '+')> */
		func() bool {
			position218, tokenIndex218 := position, tokenIndex
			{
				position219 := position
				{
					position220, tokenIndex220 := position, tokenIndex
					if buffer[position] != rune('-') {
						goto l221
					}
					position++
					goto l220
				l221:
					position, tokenIndex = position220, tokenIndex220
					if buffer[position] != rune('+') {
						goto l218
					}
					position++
				}
			l220:
				add(ruleSign, position219)
			}
			return true
		l218:
			position, tokenIndex = position218, tokenIndex218
			return false
		},
		/* 26 Integer <- <<(Sign? Unsigned)>> */
		func() bool {
			position222, tokenIndex222 := position, tokenIndex
			{
				position223 := position
				{
					position224 := position
					{
						position225, tokenIndex225 := position, tokenIndex
						if !_rules[ruleSign]() {
							goto l225
						}
						goto l226
					l225:
						position, tokenIndex = position225, tokenIndex225
					}
				l226:
					if !_rules[ruleUnsigned]() {
						goto l222
					}
					add(rulePegText, position224)
				}
				add(ruleInteger, position223)
			}
			return true
		l222:
			position, tokenIndex = position222, tokenIndex222
			return false
		},
		/* 27 Float <- <(Integer ('.' Unsigned)? (('e' / 'E') Integer)?)> */
		func() bool {
			position227, tokenIndex227 := position, tokenIndex
			{
				position228 := position
				if !_rules[ruleInteger]() {
					goto l227
				}
				{
					position229, tokenIndex229 := position, tokenIndex
					if buffer[position] != rune('.') {
						goto l229
					}
					position++
					if !_rules[ruleUnsigned]() {
						goto l229
					}
					goto l230
				l229:
					position, tokenIndex = position229, tokenIndex229
				}
			l230:
				{
					position231, tokenIndex231 := position, tokenIndex
					{
						position233, tokenIndex233 := position, tokenIndex
						if buffer[position] != rune('e') {
							goto l234
						}
						position++
						goto l233
					l234:
						position, tokenIndex = position233, tokenIndex233
						if buffer[position] != rune('E') {
							goto l231
						}
						position++
					}
				l233:
					if !_rules[ruleInteger]() {
						goto l231
					}
					goto l232
				l231:
					position, tokenIndex = position231, tokenIndex231
				}
			l232:
				add(ruleFloat, position228)
			}
			return true
		l227:
			position, tokenIndex = position227, tokenIndex227
			return false
		},
		/* 28 Identifier <- <(!Keyword <(([a-z] / [A-Z] / '_') IdChar*)>)> */
		func() bool {
			position235, tokenIndex235 := position, tokenIndex
			{
				position236 := position
				{
					position237, tokenIndex237 := position, tokenIndex
					if !_rules[ruleKeyword]() {
						goto l237
					}
					goto l235
				l237:
					position, tokenIndex = position237, tokenIndex237
				}
				{
					position238 := position
					{
						position239, tokenIndex239 := position, tokenIndex
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l240
						}
						position++
						goto l239
					l240:
						position, tokenIndex = position239, tokenIndex239
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l241
						}
						position++
						goto l239
					l241:
						position, tokenIndex = position239, tokenIndex239
						if buffer[position] != rune('_') {
							goto l235
						}
						position++
					}
				l239:
				l242:
					{
						position243, tokenIndex243 := position, tokenIndex
						if !_rules[ruleIdChar]() {
							goto l243
						}
						goto l242
					l243:
						position, tokenIndex = position243, tokenIndex243
					}
					add(rulePegText, position238)
				}
				add(ruleIdentifier, position236)
			}
			return true
		l235:
			position, tokenIndex = position235, tokenIndex235
			return false
		},
		/* 29 IdChar <- <([a-z] / [A-Z] / [0-9] / '_')> */
		func() bool {
			position244, tokenIndex244 := position, tokenIndex
			{
				position245 := position
				{
					position246, tokenIndex246 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l247
					}
					position++
					goto l246
				l247:
					position, tokenIndex = position246, tokenIndex246
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l248
					}
					position++
					goto l246
				l248:
					position, tokenIndex = position246, tokenIndex246
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l249
					}
					position++
					goto l246
				l249:
					position, tokenIndex = position246, tokenIndex246
					if buffer[position] != rune('_') {
						goto l244
					}
					position++
				}
			l246:
				add(ruleIdChar, position245)
			}
			return true
		l244:
			position, tokenIndex = position244, tokenIndex244
			return false
		},
		/* 30 Keyword <- <((('s' 'e' 'l' 'e' 'c' 't') / ('g' 'r' 'o' 'u' 'p' ' ' 'b' 'y') / ('f' 'i' 'l' 't' 'e' 'r' 's') / ('o' 'r' 'd' 'e' 'r' ' ' 'b' 'y') / ('d' 'e' 's' 'c') / ('l' 'i' 'm' 'i' 't')) !IdChar)> */
		func() bool {
			position250, tokenIndex250 := position, tokenIndex
			{
				position251 := position
				{
					position252, tokenIndex252 := position, tokenIndex
					if buffer[position] != rune('s') {
						goto l253
					}
					position++
					if buffer[position] != rune('e') {
						goto l253
					}
					position++
					if buffer[position] != rune('l') {
						goto l253
					}
					position++
					if buffer[position] != rune('e') {
						goto l253
					}
					position++
					if buffer[position] != rune('c') {
						goto l253
					}
					position++
					if buffer[position] != rune('t') {
						goto l253
					}
					position++
					goto l252
				l253:
					position, tokenIndex = position252, tokenIndex252
					if buffer[position] != rune('g') {
						goto l254
					}
					position++
					if buffer[position] != rune('r') {
						goto l254
					}
					position++
					if buffer[position] != rune('o') {
						goto l254
					}
					position++
					if buffer[position] != rune('u') {
						goto l254
					}
					position++
					if buffer[position] != rune('p') {
						goto l254
					}
					position++
					if buffer[position] != rune(' ') {
						goto l254
					}
					position++
					if buffer[position] != rune('b') {
						goto l254
					}
					position++
					if buffer[position] != rune('y') {
						goto l254
					}
					position++
					goto l252
				l254:
					position, tokenIndex = position252, tokenIndex252
					if buffer[position] != rune('f') {
						goto l255
					}
					position++
					if buffer[position] != rune('i') {
						goto l255
					}
					position++
					if buffer[position] != rune('l') {
						goto l255
					}
					position++
					if buffer[position] != rune('t') {
						goto l255
					}
					position++
					if buffer[position] != rune('e') {
						goto l255
					}
					position++
					if buffer[position] != rune('r') {
						goto l255
					}
					position++
					if buffer[position] != rune('s') {
						goto l255
					}
					position++
					goto l252
				l255:
					position, tokenIndex = position252, tokenIndex252
					if buffer[position] != rune('o') {
						goto l256
					}
					position++
					if buffer[position] != rune('r') {
						goto l256
					}
					position++
					if buffer[position] != rune('d') {
						goto l256
					}
					position++
					if buffer[position] != rune('e') {
						goto l256
					}
					position++
					if buffer[position] != rune('r') {
						goto l256
					}
					position++
					if buffer[position] != rune(' ') {
						goto l256
					}
					position++
					if buffer[position] != rune('b') {
						goto l256
					}
					position++
					if buffer[position] != rune('y') {
						goto l256
					}
					position++
					goto l252
				l256:
					position, tokenIndex = position252, tokenIndex252
					if buffer[position] != rune('d') {
						goto l257
					}
					position++
					if buffer[position] != rune('e') {
						goto l257
					}
					position++
					if buffer[position] != rune('s') {
						goto l257
					}
					position++
					if buffer[position] != rune('c') {
						goto l257
					}
					position++
					goto l252
				l257:
					position, tokenIndex = position252, tokenIndex252
					if buffer[position] != rune('l') {
						goto l250
					}
					position++
					if buffer[position] != rune('i') {
						goto l250
					}
					position++
					if buffer[position] != rune('m') {
						goto l250
					}
					position++
					if buffer[position] != rune('i') {
						goto l250
					}
					position++
					if buffer[position] != rune('t') {
						goto l250
					}
					position++
				}
			l252:
				{
					position258, tokenIndex258 := position, tokenIndex
					if !_rules[ruleIdChar]() {
						goto l258
					}
					goto l250
				l258:
					position, tokenIndex = position258, tokenIndex258
				}
				add(ruleKeyword, position251)
			}
			return true
		l250:
			position, tokenIndex = position250, tokenIndex250
			return false
		},
		/* 31 _ <- <(' ' / '\t' / ('\r' '\n') / '\n' / '\r')*> */
		func() bool {
			{
				position260 := position
			l261:
				{
					position262, tokenIndex262 := position, tokenIndex
					{
						position263, tokenIndex263 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l264
						}
						position++
						goto l263
					l264:
						position, tokenIndex = position263, tokenIndex263
						if buffer[position] != rune('\t') {
							goto l265
						}
						position++
						goto l263
					l265:
						position, tokenIndex = position263, tokenIndex263
						if buffer[position] != rune('\r') {
							goto l266
						}
						position++
						if buffer[position] != rune('\n') {
							goto l266
						}
						position++
						goto l263
					l266:
						position, tokenIndex = position263, tokenIndex263
						if buffer[position] != rune('\n') {
							goto l267
						}
						position++
						goto l263
					l267:
						position, tokenIndex = position263, tokenIndex263
						if buffer[position] != rune('\r') {
							goto l262
						}
						position++
					}
				l263:
					goto l261
				l262:
					position, tokenIndex = position262, tokenIndex262
				}
				add(rule_, position260)
			}
			return true
		},
		/* 32 LPAR <- <(_ '(' _)> */
		func() bool {
			position268, tokenIndex268 := position, tokenIndex
			{
				position269 := position
				if !_rules[rule_]() {
					goto l268
				}
				if buffer[position] != rune('(') {
					goto l268
				}
				position++
				if !_rules[rule_]() {
					goto l268
				}
				add(ruleLPAR, position269)
			}
			return true
		l268:
			position, tokenIndex = position268, tokenIndex268
			return false
		},
		/* 33 RPAR <- <(_ ')' _)> */
		func() bool {
			position270, tokenIndex270 := position, tokenIndex
			{
				position271 := position
				if !_rules[rule_]() {
					goto l270
				}
				if buffer[position] != rune(')') {
					goto l270
				}
				position++
				if !_rules[rule_]() {
					goto l270
				}
				add(ruleRPAR, position271)
			}
			return true
		l270:
			position, tokenIndex = position270, tokenIndex270
			return false
		},
		/* 34 COMMA <- <(_ ',' _)> */
		func() bool {
			position272, tokenIndex272 := position, tokenIndex
			{
				position273 := position
				if !_rules[rule_]() {
					goto l272
				}
				if buffer[position] != rune(',') {
					goto l272
				}
				position++
				if !_rules[rule_]() {
					goto l272
				}
				add(ruleCOMMA, position273)
			}
			return true
		l272:
			position, tokenIndex = position272, tokenIndex272
			return false
		},
		/* 36 Action0 <- <{ p.currentSection = "columns" }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 37 Action1 <- <{ p.currentSection = "group by" }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 38 Action2 <- <{ p.currentSection = "order by" }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		nil,
		/* 40 Action3 <- <{ p.SetLimit(text) }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 41 Action4 <- <{ p.AddColumn() }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 42 Action5 <- <{ p.SetColumnName(text) }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 43 Action6 <- <{ p.SetColumnName(text) }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 44 Action7 <- <{ p.SetColumnAggregate(text) }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 45 Action8 <- <{ p.SetColumnName(text)      }> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 46 Action9 <- <{ p.AddFilter() }> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
		/* 47 Action10 <- <{ p.SetFilterColumn(text) }> */
		func() bool {
			{
				add(ruleAction10, position)
			}
			return true
		},
		/* 48 Action11 <- <{ p.SetFilterOperator(text) }> */
		func() bool {
			{
				add(ruleAction11, position)
			}
			return true
		},
		/* 49 Action12 <- <{ p.SetFilterValueFloat(text) }> */
		func() bool {
			{
				add(ruleAction12, position)
			}
			return true
		},
		/* 50 Action13 <- <{ p.SetFilterValueInteger(text) }> */
		func() bool {
			{
				add(ruleAction13, position)
			}
			return true
		},
		/* 51 Action14 <- <{ p.SetFilterValueString(text) }> */
		func() bool {
			{
				add(ruleAction14, position)
			}
			return true
		},
		/* 52 Action15 <- <{ p.SetDescending() }> */
		func() bool {
			{
				add(ruleAction15, position)
			}
			return true
		},
	}
	p.rules = _rules
}
