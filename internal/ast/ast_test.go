package ast

import "testing"

func TestLexCondition(t *testing.T) {
	tokens := LexCondition("a AND b")
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
	}
}
