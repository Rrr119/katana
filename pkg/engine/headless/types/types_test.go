package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElementHashDistinguishesDataAttributes(t *testing.T) {
	one := &HTMLElement{TagName: "BUTTON", Attributes: map[string]string{"data-path": "/one"}, TextContent: "One"}
	two := &HTMLElement{TagName: "BUTTON", Attributes: map[string]string{"data-path": "/two"}, TextContent: "Two"}
	require.NotEqual(t, one.Hash(), two.Hash())
}

func TestElementHashFallsBackToText(t *testing.T) {
	one := &HTMLElement{TagName: "BUTTON", TextContent: "  Save\n draft "}
	two := &HTMLElement{TagName: "BUTTON", TextContent: "Publish"}
	require.NotEqual(t, one.Hash(), two.Hash())

	// Whitespace-only differences are not a new action.
	same := &HTMLElement{TagName: "BUTTON", TextContent: "Save draft"}
	require.Equal(t, one.Hash(), same.Hash())
}

func TestFormHashIncludesDistinctFormFields(t *testing.T) {
	form := func(inputName string) *HTMLForm {
		return &HTMLForm{
			TagName: "FORM",
			Action:  "/search",
			Method:  "get",
			Elements: []*HTMLElement{{
				TagName:    "INPUT",
				Attributes: map[string]string{"name": inputName},
			}},
		}
	}
	require.NotEqual(t, form("q").Hash(), form("email").Hash())
}
