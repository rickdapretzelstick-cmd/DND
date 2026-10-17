package engine

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
)

// RollResult is one evaluated dice expression such as "2d6+3".
type RollResult struct {
	Expr  string `json:"expr"`
	Rolls []int  `json:"rolls"`
	Mod   int    `json:"mod"`
	Total int    `json:"total"`
}

func (r RollResult) String() string {
	if len(r.Rolls) == 0 {
		return fmt.Sprintf("%d", r.Total)
	}
	parts := make([]string, len(r.Rolls))
	for i, v := range r.Rolls {
		parts[i] = strconv.Itoa(v)
	}
	s := "[" + strings.Join(parts, ",") + "]"
	if r.Mod > 0 {
		s += fmt.Sprintf("+%d", r.Mod)
	} else if r.Mod < 0 {
		s += fmt.Sprintf("%d", r.Mod)
	}
	return fmt.Sprintf("%s = %d", s, r.Total)
}

var diceTerm = regexp.MustCompile(`([+-]?)\s*(\d*)\s*[dD]\s*(\d+)|([+-]?)\s*(\d+)`)

// Die rolls a single n-sided die.
func Die(n int) int {
	if n < 1 {
		return 0
	}
	return rand.IntN(n) + 1
}

// D20 rolls a d20 with optional advantage (+1) or disadvantage (-1).
// It returns the kept roll and every die that was thrown.
func D20(adv int) (int, []int) {
	a := Die(20)
	if adv == 0 {
		return a, []int{a}
	}
	b := Die(20)
	if (adv > 0 && b > a) || (adv < 0 && b < a) {
		return b, []int{a, b}
	}
	return a, []int{a, b}
}

// Roll evaluates a dice expression like "1d8+2", "3d6", "d20-1" or "2d4+1d6+3".
func Roll(expr string) (RollResult, error) {
	res := RollResult{Expr: strings.TrimSpace(expr)}
	clean := strings.ReplaceAll(res.Expr, " ", "")
	if clean == "" {
		return res, fmt.Errorf("empty dice expression")
	}
	matches := diceTerm.FindAllStringSubmatchIndex(clean, -1)
	consumed := 0
	for _, m := range matches {
		if m[0] != consumed {
			return res, fmt.Errorf("can't read dice expression %q", expr)
		}
		consumed = m[1]
		sub := func(i int) string {
			if m[2*i] < 0 {
				return ""
			}
			return clean[m[2*i]:m[2*i+1]]
		}
		if sub(3) != "" { // NdS
			sign := 1
			if sub(1) == "-" {
				sign = -1
			}
			n := 1
			if sub(2) != "" {
				n, _ = strconv.Atoi(sub(2))
			}
			sides, _ := strconv.Atoi(sub(3))
			if n > 100 || sides > 1000 || sides < 1 {
				return res, fmt.Errorf("dice expression %q is too large", expr)
			}
			for i := 0; i < n; i++ {
				v := Die(sides)
				res.Rolls = append(res.Rolls, v*sign)
				res.Total += v * sign
			}
		} else {
			v, _ := strconv.Atoi(sub(5))
			if sub(4) == "-" {
				v = -v
			}
			res.Mod += v
			res.Total += v
		}
	}
	if consumed != len(clean) || len(matches) == 0 {
		return res, fmt.Errorf("can't read dice expression %q", expr)
	}
	return res, nil
}

// MustRoll rolls an expression that is known to be valid (game data).
func MustRoll(expr string) RollResult {
	r, err := Roll(expr)
	if err != nil {
		return RollResult{Expr: expr}
	}
	return r
}

// DoubleDice doubles the number of dice in an expression for critical hits: "1d8+3" -> "2d8+3".
func DoubleDice(expr string) string {
	clean := strings.ReplaceAll(expr, " ", "")
	return diceTerm.ReplaceAllStringFunc(clean, func(t string) string {
		m := diceTerm.FindStringSubmatch(t)
		if m[3] == "" {
			return t
		}
		n := 1
		if m[2] != "" {
			n, _ = strconv.Atoi(m[2])
		}
		return fmt.Sprintf("%s%dd%s", m[1], n*2, m[3])
	})
}

func pick[T any](xs []T) T { return xs[rand.IntN(len(xs))] }

func chance(p float64) bool { return rand.Float64() < p }
