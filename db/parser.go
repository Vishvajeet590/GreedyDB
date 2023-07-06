package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ParsedQuery struct {
	Type              string
	Key               string
	Value             string
	Expiry            bool
	ExpiryTime        time.Duration
	KeyExistCondition int
	ListName          string
	ListValues        []string
}

type step int

const (
	stepType step = iota
	stepSetKeyName
	stepValue
	stepExpiry
	stepExist
	stepGetKeyName
	stepQPushListName
	stepQPushListValue
	stepQPop
	stepBQPop
	stepBQPopDelay
)

var reservedWords = []string{"SET", "GET", "EX", "NX", "XX"}

type Parser struct {
	i           int
	queryTokens []string
	queryString string
	step        step
	Query       *ParsedQuery
}

func ParseCommand(queryStr string) (*ParsedQuery, error) {
	query := ParsedQuery{}
	parser := Parser{
		queryString: queryStr,
		Query:       &query,
	}
	parsedQuery, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return parsedQuery, nil
}

func (p *Parser) Parse() (*ParsedQuery, error) {

	if p.queryString == "" {
		return nil, fmt.Errorf("query is empty")
	}
	p.queryTokens = strings.Split(p.queryString, " ")
	p.step = stepType

	for {
		if p.i > len(p.queryTokens) {
			return p.Query, nil
		}
		switch p.step {
		case stepType:
			switch strings.ToUpper(p.peek()) {
			case "SET":
				p.Query.Type = "SET"
				p.step = stepSetKeyName
				p.pop()
			case "GET":
				p.Query.Type = "GET"
				p.step = stepGetKeyName
				p.pop()
			case "QPUSH":
				p.Query.Type = "QPUSH"
				p.step = stepQPushListName
				p.pop()
			case "QPOP":
				p.Query.Type = "QPOP"
				p.step = stepQPop
				p.pop()
			case "BQPOP":
				p.Query.Type = "BQPOP"
				p.step = stepBQPop
				p.pop()

			default:
				return nil, fmt.Errorf("invalid command")
			}

		case stepSetKeyName:
			if p.peek() == "" {
				return nil, fmt.Errorf("key and value needed")
			}
			for _, rWords := range reservedWords {
				if rWords == p.peek() {
					return nil, fmt.Errorf("key cannot be any of the reserved word")
				}
			}

			p.Query.Key = p.peek()
			p.step = stepValue
			p.pop()

		case stepValue:
			if p.peek() == "" {
				return nil, fmt.Errorf("value is needed")
			}
			p.Query.Value = p.peek()
			p.pop()

			if p.i >= len(p.queryTokens) {
				return p.Query, nil
			}

			if p.peek() != "" && p.peek() == "EX" {
				p.step = stepExpiry
			} else if p.peek() != "" && (p.peek() == "NX" || p.peek() == "XX") {
				p.step = stepExist
				continue
			} else {
				return nil, fmt.Errorf("invalid format")
			}

			p.pop()

		case stepExpiry:
			if p.peek() == "" {
				return nil, fmt.Errorf("expiry time is needed")
			}
			expiryTime, err := strconv.Atoi(p.peek())
			if err != nil {
				return nil, fmt.Errorf("expiry time must in integer")
			}
			p.Query.Expiry = true
			p.Query.ExpiryTime = time.Duration(expiryTime * 1000000000)
			p.pop()
			if p.i >= len(p.queryTokens) {
				return p.Query, nil
			}

			if p.peek() != "" && (p.peek() == "NX" || p.peek() == "XX") {
				p.step = stepExist
			} else {
				return nil, fmt.Errorf("invalid format")
			}

		case stepExist:
			if p.peek() == "" {
				return nil, fmt.Errorf("invalid format for expiry")
			}
			if strings.ToUpper(p.peek()) == "NX" {
				p.Query.KeyExistCondition = 0
			} else if strings.ToUpper(p.peek()) == "XX" {
				p.Query.KeyExistCondition = 1
			} else {
				return nil, fmt.Errorf("invalid format for exist stratergy")
			}
			return p.Query, nil

		case stepGetKeyName:
			if p.i != len(p.queryTokens)-2 && p.peek() == "" {
				return nil, fmt.Errorf("key needed")
			}
			for _, rWords := range reservedWords {
				if rWords == p.peek() {
					return nil, fmt.Errorf("key cannot be any of the reserved word")
				}
			}

			p.Query.Key = p.peek()
			p.pop()
			if p.i < len(p.queryTokens) {
				return nil, fmt.Errorf("invalid get format expected : Get key_name")
			}
			return p.Query, nil

		case stepQPushListName:
			if p.peek() == "" {
				return nil, fmt.Errorf("list name and values needed")
			}
			for _, rWords := range reservedWords {
				if rWords == p.peek() {
					return nil, fmt.Errorf("list name cannot be any of the reserved word")
				}
			}
			p.Query.ListName = p.peek()
			p.step = stepQPushListValue
			p.pop()

		case stepQPushListValue:
			if p.peek() == "" {
				return nil, fmt.Errorf("values needed")
			}
			values := make([]string, 0)
			for p.i < len(p.queryTokens) {
				values = append(values, p.queryTokens[p.i])
				p.pop()
			}
			p.Query.ListValues = values
			return p.Query, nil

		case stepQPop:
			if p.peek() == "" {
				return nil, fmt.Errorf("list name and values needed")
			}
			for _, rWords := range reservedWords {
				if rWords == p.peek() {
					return nil, fmt.Errorf("list name cannot be any of the reserved word")
				}
			}
			p.Query.ListName = p.peek()
			return p.Query, nil

		case stepBQPop:
			if p.peek() == "" {
				return nil, fmt.Errorf("list name is required")
			}
			for _, rWords := range reservedWords {
				if rWords == p.peek() {
					return nil, fmt.Errorf("list name cannot be any of the reserved word")
				}
			}
			p.Query.ListName = p.peek()
			p.step = stepBQPopDelay
			p.pop()

		case stepBQPopDelay:
			if p.peek() == "" {
				return nil, fmt.Errorf("mention seconds of delay")
			}
			expiryTime, err := strconv.Atoi(p.peek())
			if err != nil {
				return nil, fmt.Errorf("expiry time must in integer")
			}
			p.Query.Expiry = true
			p.Query.ExpiryTime = time.Duration(expiryTime * 1000000000)
			fmt.Printf("\vEXP = %v", p.Query.ExpiryTime)
			return p.Query, nil
		}
	}

}

func (p *Parser) pop() {
	if p.i < len(p.queryTokens) {
		p.i++
	}
}

func (p *Parser) peek() string {
	if p.i > len(p.queryTokens)-1 {
		return ""
	}

	//Handling whitespaces in between of ParsedQuery
	//Example - Set key_a    10   EX    20
	for {
		if p.queryTokens[p.i] == "" {
			p.pop()
		} else {
			break
		}
	}

	return p.queryTokens[p.i]
}
