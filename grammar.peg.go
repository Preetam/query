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
	ruleFilterCondition
	ruleFilterValue
	ruleValue
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
	"FilterCondition",
	"FilterValue",
	"Value",
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
	rules  [52]func() bool
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
			p.SetFilterCondition(text)
		case ruleAction12:
			p.SetFilterValue(text)
		case ruleAction13:
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
					{
						position4 := position
						{
							position5, tokenIndex5 := position, tokenIndex
							if buffer[position] != rune('s') {
								goto l6
							}
							position++
							goto l5
						l6:
							position, tokenIndex = position5, tokenIndex5
							if buffer[position] != rune('S') {
								goto l2
							}
							position++
						}
					l5:
						{
							position7, tokenIndex7 := position, tokenIndex
							if buffer[position] != rune('e') {
								goto l8
							}
							position++
							goto l7
						l8:
							position, tokenIndex = position7, tokenIndex7
							if buffer[position] != rune('E') {
								goto l2
							}
							position++
						}
					l7:
						{
							position9, tokenIndex9 := position, tokenIndex
							if buffer[position] != rune('l') {
								goto l10
							}
							position++
							goto l9
						l10:
							position, tokenIndex = position9, tokenIndex9
							if buffer[position] != rune('L') {
								goto l2
							}
							position++
						}
					l9:
						{
							position11, tokenIndex11 := position, tokenIndex
							if buffer[position] != rune('e') {
								goto l12
							}
							position++
							goto l11
						l12:
							position, tokenIndex = position11, tokenIndex11
							if buffer[position] != rune('E') {
								goto l2
							}
							position++
						}
					l11:
						{
							position13, tokenIndex13 := position, tokenIndex
							if buffer[position] != rune('c') {
								goto l14
							}
							position++
							goto l13
						l14:
							position, tokenIndex = position13, tokenIndex13
							if buffer[position] != rune('C') {
								goto l2
							}
							position++
						}
					l13:
						{
							position15, tokenIndex15 := position, tokenIndex
							if buffer[position] != rune('t') {
								goto l16
							}
							position++
							goto l15
						l16:
							position, tokenIndex = position15, tokenIndex15
							if buffer[position] != rune('T') {
								goto l2
							}
							position++
						}
					l15:
						if !_rules[rule_]() {
							goto l2
						}
						{
							add(ruleAction0, position)
						}
						if !_rules[ruleColumns]() {
							goto l2
						}
						add(ruleColumnExpr, position4)
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
					position18, tokenIndex18 := position, tokenIndex
					{
						position20 := position
						{
							position21, tokenIndex21 := position, tokenIndex
							if buffer[position] != rune('w') {
								goto l22
							}
							position++
							goto l21
						l22:
							position, tokenIndex = position21, tokenIndex21
							if buffer[position] != rune('W') {
								goto l18
							}
							position++
						}
					l21:
						{
							position23, tokenIndex23 := position, tokenIndex
							if buffer[position] != rune('h') {
								goto l24
							}
							position++
							goto l23
						l24:
							position, tokenIndex = position23, tokenIndex23
							if buffer[position] != rune('H') {
								goto l18
							}
							position++
						}
					l23:
						{
							position25, tokenIndex25 := position, tokenIndex
							if buffer[position] != rune('e') {
								goto l26
							}
							position++
							goto l25
						l26:
							position, tokenIndex = position25, tokenIndex25
							if buffer[position] != rune('E') {
								goto l18
							}
							position++
						}
					l25:
						{
							position27, tokenIndex27 := position, tokenIndex
							if buffer[position] != rune('r') {
								goto l28
							}
							position++
							goto l27
						l28:
							position, tokenIndex = position27, tokenIndex27
							if buffer[position] != rune('R') {
								goto l18
							}
							position++
						}
					l27:
						{
							position29, tokenIndex29 := position, tokenIndex
							if buffer[position] != rune('e') {
								goto l30
							}
							position++
							goto l29
						l30:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune('E') {
								goto l18
							}
							position++
						}
					l29:
						if !_rules[rule_]() {
							goto l18
						}
						if !_rules[ruleLogicExpr]() {
							goto l18
						}
					l31:
						{
							position32, tokenIndex32 := position, tokenIndex
							if !_rules[rule_]() {
								goto l32
							}
							{
								position33, tokenIndex33 := position, tokenIndex
								if !_rules[ruleCOMMA]() {
									goto l33
								}
								goto l34
							l33:
								position, tokenIndex = position33, tokenIndex33
							}
						l34:
							if !_rules[ruleLogicExpr]() {
								goto l32
							}
							goto l31
						l32:
							position, tokenIndex = position32, tokenIndex32
						}
						add(ruleWhereExpr, position20)
					}
					goto l19
				l18:
					position, tokenIndex = position18, tokenIndex18
				}
			l19:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position35, tokenIndex35 := position, tokenIndex
					{
						position37 := position
						{
							position38, tokenIndex38 := position, tokenIndex
							if buffer[position] != rune('g') {
								goto l39
							}
							position++
							goto l38
						l39:
							position, tokenIndex = position38, tokenIndex38
							if buffer[position] != rune('G') {
								goto l35
							}
							position++
						}
					l38:
						{
							position40, tokenIndex40 := position, tokenIndex
							if buffer[position] != rune('r') {
								goto l41
							}
							position++
							goto l40
						l41:
							position, tokenIndex = position40, tokenIndex40
							if buffer[position] != rune('R') {
								goto l35
							}
							position++
						}
					l40:
						{
							position42, tokenIndex42 := position, tokenIndex
							if buffer[position] != rune('o') {
								goto l43
							}
							position++
							goto l42
						l43:
							position, tokenIndex = position42, tokenIndex42
							if buffer[position] != rune('O') {
								goto l35
							}
							position++
						}
					l42:
						{
							position44, tokenIndex44 := position, tokenIndex
							if buffer[position] != rune('u') {
								goto l45
							}
							position++
							goto l44
						l45:
							position, tokenIndex = position44, tokenIndex44
							if buffer[position] != rune('U') {
								goto l35
							}
							position++
						}
					l44:
						{
							position46, tokenIndex46 := position, tokenIndex
							if buffer[position] != rune('p') {
								goto l47
							}
							position++
							goto l46
						l47:
							position, tokenIndex = position46, tokenIndex46
							if buffer[position] != rune('P') {
								goto l35
							}
							position++
						}
					l46:
						if buffer[position] != rune(' ') {
							goto l35
						}
						position++
						{
							position48, tokenIndex48 := position, tokenIndex
							if buffer[position] != rune('b') {
								goto l49
							}
							position++
							goto l48
						l49:
							position, tokenIndex = position48, tokenIndex48
							if buffer[position] != rune('B') {
								goto l35
							}
							position++
						}
					l48:
						{
							position50, tokenIndex50 := position, tokenIndex
							if buffer[position] != rune('y') {
								goto l51
							}
							position++
							goto l50
						l51:
							position, tokenIndex = position50, tokenIndex50
							if buffer[position] != rune('Y') {
								goto l35
							}
							position++
						}
					l50:
						if !_rules[rule_]() {
							goto l35
						}
						{
							add(ruleAction1, position)
						}
						if !_rules[ruleColumns]() {
							goto l35
						}
						add(ruleGroupExpr, position37)
					}
					goto l36
				l35:
					position, tokenIndex = position35, tokenIndex35
				}
			l36:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position53, tokenIndex53 := position, tokenIndex
					{
						position55 := position
						{
							position56, tokenIndex56 := position, tokenIndex
							if buffer[position] != rune('o') {
								goto l57
							}
							position++
							goto l56
						l57:
							position, tokenIndex = position56, tokenIndex56
							if buffer[position] != rune('O') {
								goto l53
							}
							position++
						}
					l56:
						{
							position58, tokenIndex58 := position, tokenIndex
							if buffer[position] != rune('r') {
								goto l59
							}
							position++
							goto l58
						l59:
							position, tokenIndex = position58, tokenIndex58
							if buffer[position] != rune('R') {
								goto l53
							}
							position++
						}
					l58:
						{
							position60, tokenIndex60 := position, tokenIndex
							if buffer[position] != rune('d') {
								goto l61
							}
							position++
							goto l60
						l61:
							position, tokenIndex = position60, tokenIndex60
							if buffer[position] != rune('D') {
								goto l53
							}
							position++
						}
					l60:
						{
							position62, tokenIndex62 := position, tokenIndex
							if buffer[position] != rune('e') {
								goto l63
							}
							position++
							goto l62
						l63:
							position, tokenIndex = position62, tokenIndex62
							if buffer[position] != rune('E') {
								goto l53
							}
							position++
						}
					l62:
						{
							position64, tokenIndex64 := position, tokenIndex
							if buffer[position] != rune('r') {
								goto l65
							}
							position++
							goto l64
						l65:
							position, tokenIndex = position64, tokenIndex64
							if buffer[position] != rune('R') {
								goto l53
							}
							position++
						}
					l64:
						if buffer[position] != rune(' ') {
							goto l53
						}
						position++
						{
							position66, tokenIndex66 := position, tokenIndex
							if buffer[position] != rune('b') {
								goto l67
							}
							position++
							goto l66
						l67:
							position, tokenIndex = position66, tokenIndex66
							if buffer[position] != rune('B') {
								goto l53
							}
							position++
						}
					l66:
						{
							position68, tokenIndex68 := position, tokenIndex
							if buffer[position] != rune('y') {
								goto l69
							}
							position++
							goto l68
						l69:
							position, tokenIndex = position68, tokenIndex68
							if buffer[position] != rune('Y') {
								goto l53
							}
							position++
						}
					l68:
						if !_rules[rule_]() {
							goto l53
						}
						{
							add(ruleAction2, position)
						}
						if !_rules[ruleColumns]() {
							goto l53
						}
						{
							position71, tokenIndex71 := position, tokenIndex
							{
								position73 := position
								{
									position74, tokenIndex74 := position, tokenIndex
									if buffer[position] != rune('d') {
										goto l75
									}
									position++
									goto l74
								l75:
									position, tokenIndex = position74, tokenIndex74
									if buffer[position] != rune('D') {
										goto l71
									}
									position++
								}
							l74:
								{
									position76, tokenIndex76 := position, tokenIndex
									if buffer[position] != rune('e') {
										goto l77
									}
									position++
									goto l76
								l77:
									position, tokenIndex = position76, tokenIndex76
									if buffer[position] != rune('E') {
										goto l71
									}
									position++
								}
							l76:
								{
									position78, tokenIndex78 := position, tokenIndex
									if buffer[position] != rune('s') {
										goto l79
									}
									position++
									goto l78
								l79:
									position, tokenIndex = position78, tokenIndex78
									if buffer[position] != rune('S') {
										goto l71
									}
									position++
								}
							l78:
								{
									position80, tokenIndex80 := position, tokenIndex
									if buffer[position] != rune('c') {
										goto l81
									}
									position++
									goto l80
								l81:
									position, tokenIndex = position80, tokenIndex80
									if buffer[position] != rune('C') {
										goto l71
									}
									position++
								}
							l80:
								{
									add(ruleAction13, position)
								}
								add(ruleDescending, position73)
							}
							goto l72
						l71:
							position, tokenIndex = position71, tokenIndex71
						}
					l72:
						add(ruleOrderByExpr, position55)
					}
					goto l54
				l53:
					position, tokenIndex = position53, tokenIndex53
				}
			l54:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position83, tokenIndex83 := position, tokenIndex
					{
						position85 := position
						{
							position86, tokenIndex86 := position, tokenIndex
							if buffer[position] != rune('l') {
								goto l87
							}
							position++
							goto l86
						l87:
							position, tokenIndex = position86, tokenIndex86
							if buffer[position] != rune('L') {
								goto l83
							}
							position++
						}
					l86:
						{
							position88, tokenIndex88 := position, tokenIndex
							if buffer[position] != rune('i') {
								goto l89
							}
							position++
							goto l88
						l89:
							position, tokenIndex = position88, tokenIndex88
							if buffer[position] != rune('I') {
								goto l83
							}
							position++
						}
					l88:
						{
							position90, tokenIndex90 := position, tokenIndex
							if buffer[position] != rune('m') {
								goto l91
							}
							position++
							goto l90
						l91:
							position, tokenIndex = position90, tokenIndex90
							if buffer[position] != rune('M') {
								goto l83
							}
							position++
						}
					l90:
						{
							position92, tokenIndex92 := position, tokenIndex
							if buffer[position] != rune('i') {
								goto l93
							}
							position++
							goto l92
						l93:
							position, tokenIndex = position92, tokenIndex92
							if buffer[position] != rune('I') {
								goto l83
							}
							position++
						}
					l92:
						{
							position94, tokenIndex94 := position, tokenIndex
							if buffer[position] != rune('t') {
								goto l95
							}
							position++
							goto l94
						l95:
							position, tokenIndex = position94, tokenIndex94
							if buffer[position] != rune('T') {
								goto l83
							}
							position++
						}
					l94:
						if !_rules[rule_]() {
							goto l83
						}
						{
							position96 := position
							if !_rules[ruleUnsigned]() {
								goto l83
							}
							add(rulePegText, position96)
						}
						{
							add(ruleAction3, position)
						}
						add(ruleLimitExpr, position85)
					}
					goto l84
				l83:
					position, tokenIndex = position83, tokenIndex83
				}
			l84:
				if !_rules[rule_]() {
					goto l0
				}
				{
					position98, tokenIndex98 := position, tokenIndex
					if !matchDot() {
						goto l98
					}
					goto l0
				l98:
					position, tokenIndex = position98, tokenIndex98
				}
				add(ruleQuery, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 ColumnExpr <- <(('s' / 'S') ('e' / 'E') ('l' / 'L') ('e' / 'E') ('c' / 'C') ('t' / 'T') _ Action0 Columns)> */
		nil,
		/* 2 GroupExpr <- <(('g' / 'G') ('r' / 'R') ('o' / 'O') ('u' / 'U') ('p' / 'P') ' ' ('b' / 'B') ('y' / 'Y') _ Action1 Columns)> */
		nil,
		/* 3 WhereExpr <- <(('w' / 'W') ('h' / 'H') ('e' / 'E') ('r' / 'R') ('e' / 'E') _ LogicExpr (_ COMMA? LogicExpr)*)> */
		nil,
		/* 4 OrderByExpr <- <(('o' / 'O') ('r' / 'R') ('d' / 'D') ('e' / 'E') ('r' / 'R') ' ' ('b' / 'B') ('y' / 'Y') _ Action2 Columns Descending?)> */
		nil,
		/* 5 LimitExpr <- <(('l' / 'L') ('i' / 'I') ('m' / 'M') ('i' / 'I') ('t' / 'T') _ <Unsigned> Action3)> */
		nil,
		/* 6 Columns <- <(Column (COMMA Column)*)> */
		func() bool {
			position104, tokenIndex104 := position, tokenIndex
			{
				position105 := position
				if !_rules[ruleColumn]() {
					goto l104
				}
			l106:
				{
					position107, tokenIndex107 := position, tokenIndex
					if !_rules[ruleCOMMA]() {
						goto l107
					}
					if !_rules[ruleColumn]() {
						goto l107
					}
					goto l106
				l107:
					position, tokenIndex = position107, tokenIndex107
				}
				add(ruleColumns, position105)
			}
			return true
		l104:
			position, tokenIndex = position104, tokenIndex104
			return false
		},
		/* 7 Column <- <(Action4 (ColumnAggregation / (<Identifier> _ Action5) / (<'*'> _ Action6)))> */
		func() bool {
			position108, tokenIndex108 := position, tokenIndex
			{
				position109 := position
				{
					add(ruleAction4, position)
				}
				{
					position111, tokenIndex111 := position, tokenIndex
					{
						position113 := position
						{
							position114 := position
							if !_rules[ruleIdentifier]() {
								goto l112
							}
							add(rulePegText, position114)
						}
						{
							add(ruleAction7, position)
						}
						if !_rules[ruleLPAR]() {
							goto l112
						}
						{
							position116 := position
							if !_rules[ruleIdentifier]() {
								goto l112
							}
							add(rulePegText, position116)
						}
						if !_rules[ruleRPAR]() {
							goto l112
						}
						{
							add(ruleAction8, position)
						}
						add(ruleColumnAggregation, position113)
					}
					goto l111
				l112:
					position, tokenIndex = position111, tokenIndex111
					{
						position119 := position
						if !_rules[ruleIdentifier]() {
							goto l118
						}
						add(rulePegText, position119)
					}
					if !_rules[rule_]() {
						goto l118
					}
					{
						add(ruleAction5, position)
					}
					goto l111
				l118:
					position, tokenIndex = position111, tokenIndex111
					{
						position121 := position
						if buffer[position] != rune('*') {
							goto l108
						}
						position++
						add(rulePegText, position121)
					}
					if !_rules[rule_]() {
						goto l108
					}
					{
						add(ruleAction6, position)
					}
				}
			l111:
				add(ruleColumn, position109)
			}
			return true
		l108:
			position, tokenIndex = position108, tokenIndex108
			return false
		},
		/* 8 ColumnAggregation <- <(<Identifier> Action7 LPAR <Identifier> RPAR Action8)> */
		nil,
		/* 9 LogicExpr <- <((LPAR LogicExpr RPAR) / (Action9 FilterKey _ FilterCondition _ FilterValue))> */
		func() bool {
			position124, tokenIndex124 := position, tokenIndex
			{
				position125 := position
				{
					position126, tokenIndex126 := position, tokenIndex
					if !_rules[ruleLPAR]() {
						goto l127
					}
					if !_rules[ruleLogicExpr]() {
						goto l127
					}
					if !_rules[ruleRPAR]() {
						goto l127
					}
					goto l126
				l127:
					position, tokenIndex = position126, tokenIndex126
					{
						add(ruleAction9, position)
					}
					{
						position129 := position
						{
							position130 := position
							if !_rules[ruleIdentifier]() {
								goto l124
							}
							add(rulePegText, position130)
						}
						{
							add(ruleAction10, position)
						}
						add(ruleFilterKey, position129)
					}
					if !_rules[rule_]() {
						goto l124
					}
					{
						position132 := position
						{
							position133 := position
							{
								position134 := position
								{
									position135, tokenIndex135 := position, tokenIndex
									if buffer[position] != rune('=') {
										goto l136
									}
									position++
									goto l135
								l136:
									position, tokenIndex = position135, tokenIndex135
									if buffer[position] != rune('!') {
										goto l137
									}
									position++
									if buffer[position] != rune('=') {
										goto l137
									}
									position++
									goto l135
								l137:
									position, tokenIndex = position135, tokenIndex135
									if buffer[position] != rune('<') {
										goto l138
									}
									position++
									if buffer[position] != rune('=') {
										goto l138
									}
									position++
									goto l135
								l138:
									position, tokenIndex = position135, tokenIndex135
									if buffer[position] != rune('>') {
										goto l139
									}
									position++
									if buffer[position] != rune('=') {
										goto l139
									}
									position++
									goto l135
								l139:
									position, tokenIndex = position135, tokenIndex135
									if buffer[position] != rune('<') {
										goto l140
									}
									position++
									goto l135
								l140:
									position, tokenIndex = position135, tokenIndex135
									if buffer[position] != rune('>') {
										goto l141
									}
									position++
									goto l135
								l141:
									position, tokenIndex = position135, tokenIndex135
									{
										position142, tokenIndex142 := position, tokenIndex
										if buffer[position] != rune('m') {
											goto l143
										}
										position++
										goto l142
									l143:
										position, tokenIndex = position142, tokenIndex142
										if buffer[position] != rune('M') {
											goto l124
										}
										position++
									}
								l142:
									{
										position144, tokenIndex144 := position, tokenIndex
										if buffer[position] != rune('a') {
											goto l145
										}
										position++
										goto l144
									l145:
										position, tokenIndex = position144, tokenIndex144
										if buffer[position] != rune('A') {
											goto l124
										}
										position++
									}
								l144:
									{
										position146, tokenIndex146 := position, tokenIndex
										if buffer[position] != rune('t') {
											goto l147
										}
										position++
										goto l146
									l147:
										position, tokenIndex = position146, tokenIndex146
										if buffer[position] != rune('T') {
											goto l124
										}
										position++
									}
								l146:
									{
										position148, tokenIndex148 := position, tokenIndex
										if buffer[position] != rune('c') {
											goto l149
										}
										position++
										goto l148
									l149:
										position, tokenIndex = position148, tokenIndex148
										if buffer[position] != rune('C') {
											goto l124
										}
										position++
									}
								l148:
									{
										position150, tokenIndex150 := position, tokenIndex
										if buffer[position] != rune('h') {
											goto l151
										}
										position++
										goto l150
									l151:
										position, tokenIndex = position150, tokenIndex150
										if buffer[position] != rune('H') {
											goto l124
										}
										position++
									}
								l150:
									{
										position152, tokenIndex152 := position, tokenIndex
										if buffer[position] != rune('e') {
											goto l153
										}
										position++
										goto l152
									l153:
										position, tokenIndex = position152, tokenIndex152
										if buffer[position] != rune('E') {
											goto l124
										}
										position++
									}
								l152:
									{
										position154, tokenIndex154 := position, tokenIndex
										if buffer[position] != rune('s') {
											goto l155
										}
										position++
										goto l154
									l155:
										position, tokenIndex = position154, tokenIndex154
										if buffer[position] != rune('S') {
											goto l124
										}
										position++
									}
								l154:
								}
							l135:
								add(ruleOPERATOR, position134)
							}
							add(rulePegText, position133)
						}
						{
							add(ruleAction11, position)
						}
						add(ruleFilterCondition, position132)
					}
					if !_rules[rule_]() {
						goto l124
					}
					{
						position157 := position
						{
							position158 := position
							{
								position159 := position
								{
									position160, tokenIndex160 := position, tokenIndex
									{
										position162 := position
										if !_rules[ruleInteger]() {
											goto l161
										}
										{
											position163, tokenIndex163 := position, tokenIndex
											if buffer[position] != rune('.') {
												goto l163
											}
											position++
											if !_rules[ruleUnsigned]() {
												goto l163
											}
											goto l164
										l163:
											position, tokenIndex = position163, tokenIndex163
										}
									l164:
										{
											position165, tokenIndex165 := position, tokenIndex
											{
												position167, tokenIndex167 := position, tokenIndex
												if buffer[position] != rune('e') {
													goto l168
												}
												position++
												goto l167
											l168:
												position, tokenIndex = position167, tokenIndex167
												if buffer[position] != rune('E') {
													goto l165
												}
												position++
											}
										l167:
											if !_rules[ruleInteger]() {
												goto l165
											}
											goto l166
										l165:
											position, tokenIndex = position165, tokenIndex165
										}
									l166:
										add(ruleFloat, position162)
									}
									goto l160
								l161:
									position, tokenIndex = position160, tokenIndex160
									if !_rules[ruleInteger]() {
										goto l169
									}
									goto l160
								l169:
									position, tokenIndex = position160, tokenIndex160
									{
										position170 := position
										if buffer[position] != rune('"') {
											goto l124
										}
										position++
										{
											position173 := position
										l174:
											{
												position175, tokenIndex175 := position, tokenIndex
												{
													position176 := position
													{
														position177, tokenIndex177 := position, tokenIndex
														{
															position179 := position
															{
																position180, tokenIndex180 := position, tokenIndex
																{
																	position182 := position
																	if buffer[position] != rune('\\') {
																		goto l181
																	}
																	position++
																	{
																		position183, tokenIndex183 := position, tokenIndex
																		if buffer[position] != rune('\'') {
																			goto l184
																		}
																		position++
																		goto l183
																	l184:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('"') {
																			goto l185
																		}
																		position++
																		goto l183
																	l185:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('?') {
																			goto l186
																		}
																		position++
																		goto l183
																	l186:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('\\') {
																			goto l187
																		}
																		position++
																		goto l183
																	l187:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('a') {
																			goto l188
																		}
																		position++
																		goto l183
																	l188:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('b') {
																			goto l189
																		}
																		position++
																		goto l183
																	l189:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('f') {
																			goto l190
																		}
																		position++
																		goto l183
																	l190:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('n') {
																			goto l191
																		}
																		position++
																		goto l183
																	l191:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('r') {
																			goto l192
																		}
																		position++
																		goto l183
																	l192:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('t') {
																			goto l193
																		}
																		position++
																		goto l183
																	l193:
																		position, tokenIndex = position183, tokenIndex183
																		if buffer[position] != rune('v') {
																			goto l181
																		}
																		position++
																	}
																l183:
																	add(ruleSimpleEscape, position182)
																}
																goto l180
															l181:
																position, tokenIndex = position180, tokenIndex180
																{
																	position195 := position
																	if buffer[position] != rune('\\') {
																		goto l194
																	}
																	position++
																	if c := buffer[position]; c < rune('0') || c > rune('7') {
																		goto l194
																	}
																	position++
																	{
																		position196, tokenIndex196 := position, tokenIndex
																		if c := buffer[position]; c < rune('0') || c > rune('7') {
																			goto l196
																		}
																		position++
																		goto l197
																	l196:
																		position, tokenIndex = position196, tokenIndex196
																	}
																l197:
																	{
																		position198, tokenIndex198 := position, tokenIndex
																		if c := buffer[position]; c < rune('0') || c > rune('7') {
																			goto l198
																		}
																		position++
																		goto l199
																	l198:
																		position, tokenIndex = position198, tokenIndex198
																	}
																l199:
																	add(ruleOctalEscape, position195)
																}
																goto l180
															l194:
																position, tokenIndex = position180, tokenIndex180
																{
																	position201 := position
																	if buffer[position] != rune('\\') {
																		goto l200
																	}
																	position++
																	if buffer[position] != rune('x') {
																		goto l200
																	}
																	position++
																	if !_rules[ruleHexDigit]() {
																		goto l200
																	}
																l202:
																	{
																		position203, tokenIndex203 := position, tokenIndex
																		if !_rules[ruleHexDigit]() {
																			goto l203
																		}
																		goto l202
																	l203:
																		position, tokenIndex = position203, tokenIndex203
																	}
																	add(ruleHexEscape, position201)
																}
																goto l180
															l200:
																position, tokenIndex = position180, tokenIndex180
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
																			goto l178
																		}
																		position++
																		if buffer[position] != rune('U') {
																			goto l178
																		}
																		position++
																		if !_rules[ruleHexQuad]() {
																			goto l178
																		}
																		if !_rules[ruleHexQuad]() {
																			goto l178
																		}
																	}
																l205:
																	add(ruleUniversalCharacter, position204)
																}
															}
														l180:
															add(ruleEscape, position179)
														}
														goto l177
													l178:
														position, tokenIndex = position177, tokenIndex177
														{
															position207, tokenIndex207 := position, tokenIndex
															{
																position208, tokenIndex208 := position, tokenIndex
																if buffer[position] != rune('"') {
																	goto l209
																}
																position++
																goto l208
															l209:
																position, tokenIndex = position208, tokenIndex208
																if buffer[position] != rune('\n') {
																	goto l210
																}
																position++
																goto l208
															l210:
																position, tokenIndex = position208, tokenIndex208
																if buffer[position] != rune('\\') {
																	goto l207
																}
																position++
															}
														l208:
															goto l175
														l207:
															position, tokenIndex = position207, tokenIndex207
														}
														if !matchDot() {
															goto l175
														}
													}
												l177:
													add(ruleStringChar, position176)
												}
												goto l174
											l175:
												position, tokenIndex = position175, tokenIndex175
											}
											add(rulePegText, position173)
										}
										if buffer[position] != rune('"') {
											goto l124
										}
										position++
									l171:
										{
											position172, tokenIndex172 := position, tokenIndex
											if buffer[position] != rune('"') {
												goto l172
											}
											position++
											{
												position211 := position
											l212:
												{
													position213, tokenIndex213 := position, tokenIndex
													{
														position214 := position
														{
															position215, tokenIndex215 := position, tokenIndex
															{
																position217 := position
																{
																	position218, tokenIndex218 := position, tokenIndex
																	{
																		position220 := position
																		if buffer[position] != rune('\\') {
																			goto l219
																		}
																		position++
																		{
																			position221, tokenIndex221 := position, tokenIndex
																			if buffer[position] != rune('\'') {
																				goto l222
																			}
																			position++
																			goto l221
																		l222:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('"') {
																				goto l223
																			}
																			position++
																			goto l221
																		l223:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('?') {
																				goto l224
																			}
																			position++
																			goto l221
																		l224:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('\\') {
																				goto l225
																			}
																			position++
																			goto l221
																		l225:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('a') {
																				goto l226
																			}
																			position++
																			goto l221
																		l226:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('b') {
																				goto l227
																			}
																			position++
																			goto l221
																		l227:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('f') {
																				goto l228
																			}
																			position++
																			goto l221
																		l228:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('n') {
																				goto l229
																			}
																			position++
																			goto l221
																		l229:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('r') {
																				goto l230
																			}
																			position++
																			goto l221
																		l230:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('t') {
																				goto l231
																			}
																			position++
																			goto l221
																		l231:
																			position, tokenIndex = position221, tokenIndex221
																			if buffer[position] != rune('v') {
																				goto l219
																			}
																			position++
																		}
																	l221:
																		add(ruleSimpleEscape, position220)
																	}
																	goto l218
																l219:
																	position, tokenIndex = position218, tokenIndex218
																	{
																		position233 := position
																		if buffer[position] != rune('\\') {
																			goto l232
																		}
																		position++
																		if c := buffer[position]; c < rune('0') || c > rune('7') {
																			goto l232
																		}
																		position++
																		{
																			position234, tokenIndex234 := position, tokenIndex
																			if c := buffer[position]; c < rune('0') || c > rune('7') {
																				goto l234
																			}
																			position++
																			goto l235
																		l234:
																			position, tokenIndex = position234, tokenIndex234
																		}
																	l235:
																		{
																			position236, tokenIndex236 := position, tokenIndex
																			if c := buffer[position]; c < rune('0') || c > rune('7') {
																				goto l236
																			}
																			position++
																			goto l237
																		l236:
																			position, tokenIndex = position236, tokenIndex236
																		}
																	l237:
																		add(ruleOctalEscape, position233)
																	}
																	goto l218
																l232:
																	position, tokenIndex = position218, tokenIndex218
																	{
																		position239 := position
																		if buffer[position] != rune('\\') {
																			goto l238
																		}
																		position++
																		if buffer[position] != rune('x') {
																			goto l238
																		}
																		position++
																		if !_rules[ruleHexDigit]() {
																			goto l238
																		}
																	l240:
																		{
																			position241, tokenIndex241 := position, tokenIndex
																			if !_rules[ruleHexDigit]() {
																				goto l241
																			}
																			goto l240
																		l241:
																			position, tokenIndex = position241, tokenIndex241
																		}
																		add(ruleHexEscape, position239)
																	}
																	goto l218
																l238:
																	position, tokenIndex = position218, tokenIndex218
																	{
																		position242 := position
																		{
																			position243, tokenIndex243 := position, tokenIndex
																			if buffer[position] != rune('\\') {
																				goto l244
																			}
																			position++
																			if buffer[position] != rune('u') {
																				goto l244
																			}
																			position++
																			if !_rules[ruleHexQuad]() {
																				goto l244
																			}
																			goto l243
																		l244:
																			position, tokenIndex = position243, tokenIndex243
																			if buffer[position] != rune('\\') {
																				goto l216
																			}
																			position++
																			if buffer[position] != rune('U') {
																				goto l216
																			}
																			position++
																			if !_rules[ruleHexQuad]() {
																				goto l216
																			}
																			if !_rules[ruleHexQuad]() {
																				goto l216
																			}
																		}
																	l243:
																		add(ruleUniversalCharacter, position242)
																	}
																}
															l218:
																add(ruleEscape, position217)
															}
															goto l215
														l216:
															position, tokenIndex = position215, tokenIndex215
															{
																position245, tokenIndex245 := position, tokenIndex
																{
																	position246, tokenIndex246 := position, tokenIndex
																	if buffer[position] != rune('"') {
																		goto l247
																	}
																	position++
																	goto l246
																l247:
																	position, tokenIndex = position246, tokenIndex246
																	if buffer[position] != rune('\n') {
																		goto l248
																	}
																	position++
																	goto l246
																l248:
																	position, tokenIndex = position246, tokenIndex246
																	if buffer[position] != rune('\\') {
																		goto l245
																	}
																	position++
																}
															l246:
																goto l213
															l245:
																position, tokenIndex = position245, tokenIndex245
															}
															if !matchDot() {
																goto l213
															}
														}
													l215:
														add(ruleStringChar, position214)
													}
													goto l212
												l213:
													position, tokenIndex = position213, tokenIndex213
												}
												add(rulePegText, position211)
											}
											if buffer[position] != rune('"') {
												goto l172
											}
											position++
											goto l171
										l172:
											position, tokenIndex = position172, tokenIndex172
										}
										add(ruleString, position170)
									}
								}
							l160:
								add(ruleValue, position159)
							}
							add(rulePegText, position158)
						}
						{
							add(ruleAction12, position)
						}
						add(ruleFilterValue, position157)
					}
				}
			l126:
				add(ruleLogicExpr, position125)
			}
			return true
		l124:
			position, tokenIndex = position124, tokenIndex124
			return false
		},
		/* 10 OPERATOR <- <('=' / ('!' '=') / ('<' '=') / ('>' '=') / '<' / '>' / (('m' / 'M') ('a' / 'A') ('t' / 'T') ('c' / 'C') ('h' / 'H') ('e' / 'E') ('s' / 'S')))> */
		nil,
		/* 11 FilterKey <- <(<Identifier> Action10)> */
		nil,
		/* 12 FilterCondition <- <(<OPERATOR> Action11)> */
		nil,
		/* 13 FilterValue <- <(<Value> Action12)> */
		nil,
		/* 14 Value <- <(Float / Integer / String)> */
		nil,
		/* 15 Descending <- <(('d' / 'D') ('e' / 'E') ('s' / 'S') ('c' / 'C') Action13)> */
		nil,
		/* 16 String <- <('"' <StringChar*> '"')+> */
		nil,
		/* 17 StringChar <- <(Escape / (!('"' / '\n' / '\\') .))> */
		nil,
		/* 18 Escape <- <(SimpleEscape / OctalEscape / HexEscape / UniversalCharacter)> */
		nil,
		/* 19 SimpleEscape <- <('\\' ('\'' / '"' / '?' / '\\' / 'a' / 'b' / 'f' / 'n' / 'r' / 't' / 'v'))> */
		nil,
		/* 20 OctalEscape <- <('\\' [0-7] [0-7]? [0-7]?)> */
		nil,
		/* 21 HexEscape <- <('\\' 'x' HexDigit+)> */
		nil,
		/* 22 UniversalCharacter <- <(('\\' 'u' HexQuad) / ('\\' 'U' HexQuad HexQuad))> */
		nil,
		/* 23 HexQuad <- <(HexDigit HexDigit HexDigit HexDigit)> */
		func() bool {
			position263, tokenIndex263 := position, tokenIndex
			{
				position264 := position
				if !_rules[ruleHexDigit]() {
					goto l263
				}
				if !_rules[ruleHexDigit]() {
					goto l263
				}
				if !_rules[ruleHexDigit]() {
					goto l263
				}
				if !_rules[ruleHexDigit]() {
					goto l263
				}
				add(ruleHexQuad, position264)
			}
			return true
		l263:
			position, tokenIndex = position263, tokenIndex263
			return false
		},
		/* 24 HexDigit <- <([a-f] / [A-F] / [0-9])> */
		func() bool {
			position265, tokenIndex265 := position, tokenIndex
			{
				position266 := position
				{
					position267, tokenIndex267 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('f') {
						goto l268
					}
					position++
					goto l267
				l268:
					position, tokenIndex = position267, tokenIndex267
					if c := buffer[position]; c < rune('A') || c > rune('F') {
						goto l269
					}
					position++
					goto l267
				l269:
					position, tokenIndex = position267, tokenIndex267
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l265
					}
					position++
				}
			l267:
				add(ruleHexDigit, position266)
			}
			return true
		l265:
			position, tokenIndex = position265, tokenIndex265
			return false
		},
		/* 25 Unsigned <- <[0-9]+> */
		func() bool {
			position270, tokenIndex270 := position, tokenIndex
			{
				position271 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l270
				}
				position++
			l272:
				{
					position273, tokenIndex273 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l273
					}
					position++
					goto l272
				l273:
					position, tokenIndex = position273, tokenIndex273
				}
				add(ruleUnsigned, position271)
			}
			return true
		l270:
			position, tokenIndex = position270, tokenIndex270
			return false
		},
		/* 26 Sign <- <('-' / '+')> */
		nil,
		/* 27 Integer <- <<(Sign? Unsigned)>> */
		func() bool {
			position275, tokenIndex275 := position, tokenIndex
			{
				position276 := position
				{
					position277 := position
					{
						position278, tokenIndex278 := position, tokenIndex
						{
							position280 := position
							{
								position281, tokenIndex281 := position, tokenIndex
								if buffer[position] != rune('-') {
									goto l282
								}
								position++
								goto l281
							l282:
								position, tokenIndex = position281, tokenIndex281
								if buffer[position] != rune('+') {
									goto l278
								}
								position++
							}
						l281:
							add(ruleSign, position280)
						}
						goto l279
					l278:
						position, tokenIndex = position278, tokenIndex278
					}
				l279:
					if !_rules[ruleUnsigned]() {
						goto l275
					}
					add(rulePegText, position277)
				}
				add(ruleInteger, position276)
			}
			return true
		l275:
			position, tokenIndex = position275, tokenIndex275
			return false
		},
		/* 28 Float <- <(Integer ('.' Unsigned)? (('e' / 'E') Integer)?)> */
		nil,
		/* 29 Identifier <- <(!Keyword <(([a-z] / [A-Z] / '_') IdChar*)>)> */
		func() bool {
			position284, tokenIndex284 := position, tokenIndex
			{
				position285 := position
				{
					position286, tokenIndex286 := position, tokenIndex
					{
						position287 := position
						{
							position288, tokenIndex288 := position, tokenIndex
							if buffer[position] != rune('s') {
								goto l289
							}
							position++
							if buffer[position] != rune('e') {
								goto l289
							}
							position++
							if buffer[position] != rune('l') {
								goto l289
							}
							position++
							if buffer[position] != rune('e') {
								goto l289
							}
							position++
							if buffer[position] != rune('c') {
								goto l289
							}
							position++
							if buffer[position] != rune('t') {
								goto l289
							}
							position++
							goto l288
						l289:
							position, tokenIndex = position288, tokenIndex288
							if buffer[position] != rune('g') {
								goto l290
							}
							position++
							if buffer[position] != rune('r') {
								goto l290
							}
							position++
							if buffer[position] != rune('o') {
								goto l290
							}
							position++
							if buffer[position] != rune('u') {
								goto l290
							}
							position++
							if buffer[position] != rune('p') {
								goto l290
							}
							position++
							if buffer[position] != rune(' ') {
								goto l290
							}
							position++
							if buffer[position] != rune('b') {
								goto l290
							}
							position++
							if buffer[position] != rune('y') {
								goto l290
							}
							position++
							goto l288
						l290:
							position, tokenIndex = position288, tokenIndex288
							if buffer[position] != rune('f') {
								goto l291
							}
							position++
							if buffer[position] != rune('i') {
								goto l291
							}
							position++
							if buffer[position] != rune('l') {
								goto l291
							}
							position++
							if buffer[position] != rune('t') {
								goto l291
							}
							position++
							if buffer[position] != rune('e') {
								goto l291
							}
							position++
							if buffer[position] != rune('r') {
								goto l291
							}
							position++
							if buffer[position] != rune('s') {
								goto l291
							}
							position++
							goto l288
						l291:
							position, tokenIndex = position288, tokenIndex288
							if buffer[position] != rune('o') {
								goto l292
							}
							position++
							if buffer[position] != rune('r') {
								goto l292
							}
							position++
							if buffer[position] != rune('d') {
								goto l292
							}
							position++
							if buffer[position] != rune('e') {
								goto l292
							}
							position++
							if buffer[position] != rune('r') {
								goto l292
							}
							position++
							if buffer[position] != rune(' ') {
								goto l292
							}
							position++
							if buffer[position] != rune('b') {
								goto l292
							}
							position++
							if buffer[position] != rune('y') {
								goto l292
							}
							position++
							goto l288
						l292:
							position, tokenIndex = position288, tokenIndex288
							if buffer[position] != rune('d') {
								goto l293
							}
							position++
							if buffer[position] != rune('e') {
								goto l293
							}
							position++
							if buffer[position] != rune('s') {
								goto l293
							}
							position++
							if buffer[position] != rune('c') {
								goto l293
							}
							position++
							goto l288
						l293:
							position, tokenIndex = position288, tokenIndex288
							if buffer[position] != rune('l') {
								goto l286
							}
							position++
							if buffer[position] != rune('i') {
								goto l286
							}
							position++
							if buffer[position] != rune('m') {
								goto l286
							}
							position++
							if buffer[position] != rune('i') {
								goto l286
							}
							position++
							if buffer[position] != rune('t') {
								goto l286
							}
							position++
						}
					l288:
						{
							position294, tokenIndex294 := position, tokenIndex
							if !_rules[ruleIdChar]() {
								goto l294
							}
							goto l286
						l294:
							position, tokenIndex = position294, tokenIndex294
						}
						add(ruleKeyword, position287)
					}
					goto l284
				l286:
					position, tokenIndex = position286, tokenIndex286
				}
				{
					position295 := position
					{
						position296, tokenIndex296 := position, tokenIndex
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l297
						}
						position++
						goto l296
					l297:
						position, tokenIndex = position296, tokenIndex296
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l298
						}
						position++
						goto l296
					l298:
						position, tokenIndex = position296, tokenIndex296
						if buffer[position] != rune('_') {
							goto l284
						}
						position++
					}
				l296:
				l299:
					{
						position300, tokenIndex300 := position, tokenIndex
						if !_rules[ruleIdChar]() {
							goto l300
						}
						goto l299
					l300:
						position, tokenIndex = position300, tokenIndex300
					}
					add(rulePegText, position295)
				}
				add(ruleIdentifier, position285)
			}
			return true
		l284:
			position, tokenIndex = position284, tokenIndex284
			return false
		},
		/* 30 IdChar <- <([a-z] / [A-Z] / [0-9] / '_')> */
		func() bool {
			position301, tokenIndex301 := position, tokenIndex
			{
				position302 := position
				{
					position303, tokenIndex303 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l304
					}
					position++
					goto l303
				l304:
					position, tokenIndex = position303, tokenIndex303
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l305
					}
					position++
					goto l303
				l305:
					position, tokenIndex = position303, tokenIndex303
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l306
					}
					position++
					goto l303
				l306:
					position, tokenIndex = position303, tokenIndex303
					if buffer[position] != rune('_') {
						goto l301
					}
					position++
				}
			l303:
				add(ruleIdChar, position302)
			}
			return true
		l301:
			position, tokenIndex = position301, tokenIndex301
			return false
		},
		/* 31 Keyword <- <((('s' 'e' 'l' 'e' 'c' 't') / ('g' 'r' 'o' 'u' 'p' ' ' 'b' 'y') / ('f' 'i' 'l' 't' 'e' 'r' 's') / ('o' 'r' 'd' 'e' 'r' ' ' 'b' 'y') / ('d' 'e' 's' 'c') / ('l' 'i' 'm' 'i' 't')) !IdChar)> */
		nil,
		/* 32 _ <- <(' ' / '\t' / ('\r' '\n') / '\n' / '\r')*> */
		func() bool {
			{
				position309 := position
			l310:
				{
					position311, tokenIndex311 := position, tokenIndex
					{
						position312, tokenIndex312 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l313
						}
						position++
						goto l312
					l313:
						position, tokenIndex = position312, tokenIndex312
						if buffer[position] != rune('\t') {
							goto l314
						}
						position++
						goto l312
					l314:
						position, tokenIndex = position312, tokenIndex312
						if buffer[position] != rune('\r') {
							goto l315
						}
						position++
						if buffer[position] != rune('\n') {
							goto l315
						}
						position++
						goto l312
					l315:
						position, tokenIndex = position312, tokenIndex312
						if buffer[position] != rune('\n') {
							goto l316
						}
						position++
						goto l312
					l316:
						position, tokenIndex = position312, tokenIndex312
						if buffer[position] != rune('\r') {
							goto l311
						}
						position++
					}
				l312:
					goto l310
				l311:
					position, tokenIndex = position311, tokenIndex311
				}
				add(rule_, position309)
			}
			return true
		},
		/* 33 LPAR <- <(_ '(' _)> */
		func() bool {
			position317, tokenIndex317 := position, tokenIndex
			{
				position318 := position
				if !_rules[rule_]() {
					goto l317
				}
				if buffer[position] != rune('(') {
					goto l317
				}
				position++
				if !_rules[rule_]() {
					goto l317
				}
				add(ruleLPAR, position318)
			}
			return true
		l317:
			position, tokenIndex = position317, tokenIndex317
			return false
		},
		/* 34 RPAR <- <(_ ')' _)> */
		func() bool {
			position319, tokenIndex319 := position, tokenIndex
			{
				position320 := position
				if !_rules[rule_]() {
					goto l319
				}
				if buffer[position] != rune(')') {
					goto l319
				}
				position++
				if !_rules[rule_]() {
					goto l319
				}
				add(ruleRPAR, position320)
			}
			return true
		l319:
			position, tokenIndex = position319, tokenIndex319
			return false
		},
		/* 35 COMMA <- <(_ ',' _)> */
		func() bool {
			position321, tokenIndex321 := position, tokenIndex
			{
				position322 := position
				if !_rules[rule_]() {
					goto l321
				}
				if buffer[position] != rune(',') {
					goto l321
				}
				position++
				if !_rules[rule_]() {
					goto l321
				}
				add(ruleCOMMA, position322)
			}
			return true
		l321:
			position, tokenIndex = position321, tokenIndex321
			return false
		},
		/* 37 Action0 <- <{ p.currentSection = "columns" }> */
		nil,
		/* 38 Action1 <- <{ p.currentSection = "group by" }> */
		nil,
		/* 39 Action2 <- <{ p.currentSection = "order by" }> */
		nil,
		nil,
		/* 41 Action3 <- <{ p.SetLimit(text) }> */
		nil,
		/* 42 Action4 <- <{ p.AddColumn() }> */
		nil,
		/* 43 Action5 <- <{ p.SetColumnName(text) }> */
		nil,
		/* 44 Action6 <- <{ p.SetColumnName(text) }> */
		nil,
		/* 45 Action7 <- <{ p.SetColumnAggregate(text) }> */
		nil,
		/* 46 Action8 <- <{ p.SetColumnName(text)      }> */
		nil,
		/* 47 Action9 <- <{ p.AddFilter() }> */
		nil,
		/* 48 Action10 <- <{ p.SetFilterColumn(text) }> */
		nil,
		/* 49 Action11 <- <{ p.SetFilterCondition(text) }> */
		nil,
		/* 50 Action12 <- <{ p.SetFilterValue(text) }> */
		nil,
		/* 51 Action13 <- <{ p.SetDescending() }> */
		nil,
	}
	p.rules = _rules
}
