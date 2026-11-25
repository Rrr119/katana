package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/projectdiscovery/katana/pkg/engine/headless/browser"
	"github.com/projectdiscovery/katana/pkg/engine/headless/types"
	utilsformfill "github.com/projectdiscovery/katana/pkg/utils"
	"github.com/projectdiscovery/rod"
	"github.com/projectdiscovery/rod/lib/proto"
	mapsutil "github.com/projectdiscovery/utils/maps"
)

func deriveName(e *types.HTMLElement) string {
	if n, ok := e.Attributes["name"]; ok && n != "" {
		return n
	}
	return e.ID
}

func copyAttrs(src map[string]string, skipKeys ...string) mapsutil.OrderedMap[string, string] {
	skip := map[string]struct{}{}
	for _, k := range skipKeys {
		skip[k] = struct{}{}
	}
	dst := mapsutil.NewOrderedMap[string, string]()
	for k, v := range src {
		if _, s := skip[k]; !s {
			dst.Set(k, v)
		}
	}
	return dst
}

func convertHTMLElementToFormInput(element *types.HTMLElement) utilsformfill.FormInput {
	return utilsformfill.FormInput{
		Name:       deriveName(element),
		Type:       element.Type,
		Value:      element.Value,
		Attributes: copyAttrs(element.Attributes, "name", "value", "type"),
	}
}

func convertHTMLElementToFormTextArea(element *types.HTMLElement) utilsformfill.FormTextArea {
	return utilsformfill.FormTextArea{
		Name:       deriveName(element),
		Attributes: copyAttrs(element.Attributes, "name"),
	}
}

func convertHTMLElementToFormSelect(element *types.HTMLElement) utilsformfill.FormSelect {
	return utilsformfill.FormSelect{
		Name:          deriveName(element),
		Attributes:    copyAttrs(element.Attributes, "name"),
		SelectOptions: []utilsformfill.SelectOption{},
	}
}

func (c *Crawler) processForm(ctx context.Context, page *browser.BrowserPage, form *types.HTMLForm) error {
	if !c.options.AutomaticFormFill {
		return nil
	}

	var formFields []interface{}
	var submitButton *rod.Element
	elementMap := make(map[string]*rod.Element)
	var formElement *rod.Element
	if form.XPath != "" {
		element, err := page.ElementX(form.XPath)
		if err != nil {
			c.logger.Debug("Could not find form element",
				slog.String("xpath", form.XPath),
				slog.String("error", err.Error()),
			)
		} else {
			formElement = element
		}
	}

	for _, field := range form.Elements {
		if field.XPath == "" {
			continue
		}

		element, err := page.ElementX(field.XPath)
		if err != nil {
			c.logger.Debug("Could not find form element",
				slog.String("xpath", field.XPath),
				slog.String("error", err.Error()),
			)
			continue
		}

		fieldName := c.getFieldName(field)

		switch field.TagName {
		case "INPUT":
			if field.Type == "submit" || field.Type == "image" || field.Type == "button" {
				if submitButton == nil && (field.Type == "submit" || field.Type == "image") {
					submitButton = element
				}
				continue
			}

			formInput := convertHTMLElementToFormInput(field)
			formFields = append(formFields, formInput)
			if fieldName != "" {
				elementMap[fieldName] = element
			}

		case "TEXTAREA":
			formTextArea := convertHTMLElementToFormTextArea(field)
			formFields = append(formFields, formTextArea)
			if fieldName != "" {
				elementMap[fieldName] = element
			}

		case "SELECT":
			formSelect := c.buildFormSelectWithOptions(page, field, element)
			formFields = append(formFields, formSelect)
			if fieldName != "" {
				elementMap[fieldName] = element
			}

		case "BUTTON":
			if field.Type == "submit" && submitButton == nil {
				submitButton = element
			}
		}
	}

	fillSuggestions := utilsformfill.FormFillSuggestionsWithDecider(ctx, c.options.FormDecider, formFields)
	if fillSuggestions.Len() == 0 && submitButton == nil {
		return nil
	}

	if err := c.applyFormSuggestions(fillSuggestions, elementMap); err != nil {
		c.logger.Debug("Error applying form suggestions", slog.String("error", err.Error()))
	}

	if submitButton != nil {
		if err := submitButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return err
		}
		return nil
	}

	// A form does not need an explicit submit control: pressing Enter in a
	// text field submits it. requestSubmit() preserves submit listeners that
	// SPAs rely on, unlike form.submit().
	if formElement != nil {
		if _, err := formElement.Eval(`() => {
			if (typeof this.requestSubmit === "function") {
				this.requestSubmit();
			} else {
				this.submit();
			}
		}`); err != nil {
			return err
		}
	}

	return nil
}

func (c *Crawler) getFieldName(field *types.HTMLElement) string {
	return deriveName(field)
}

func (c *Crawler) buildFormSelectWithOptions(page *browser.BrowserPage, field *types.HTMLElement, element *rod.Element) utilsformfill.FormSelect {
	formSelect := convertHTMLElementToFormSelect(field)

	options, err := element.Elements("option")
	if err == nil && len(options) > 0 {
		formSelect.SelectOptions = []utilsformfill.SelectOption{}
		for _, opt := range options {
			optionValue, _ := opt.Attribute("value")
			if optionValue == nil {
				text, _ := opt.Text()
				optionValue = &text
			}

			selected, _ := opt.Attribute("selected")
			selectOption := utilsformfill.SelectOption{
				Value:    *optionValue,
				Selected: "",
			}
			if selected != nil {
				selectOption.Selected = "selected"
			}
			formSelect.SelectOptions = append(formSelect.SelectOptions, selectOption)
		}
	} else {
		formSelect.SelectOptions = []utilsformfill.SelectOption{
			{Value: utilsformfill.FormData.Placeholder, Selected: "selected"},
		}
	}

	return formSelect
}

func (c *Crawler) applyFormSuggestions(suggestions mapsutil.OrderedMap[string, string], elementMap map[string]*rod.Element) error {
	suggestions.Iterate(func(fieldName, value string) bool {
		element, exists := elementMap[fieldName]
		if !exists || value == "" {
			return true
		}

		tagName, err := element.Eval(`() => this.tagName`)
		if err != nil {
			c.logger.Debug("Failed to get element tag",
				slog.String("field", fieldName),
				slog.String("error", err.Error()),
			)
			return true
		}

		switch tagName.Value.String() {
		case "INPUT":
			inputType := "text"
			if attribute, _ := element.Attribute("type"); attribute != nil && strings.TrimSpace(*attribute) != "" {
				inputType = strings.ToLower(strings.TrimSpace(*attribute))
			}
			switch inputType {
			case "checkbox", "radio":
				controlValue := "on"
				if attribute, _ := element.Attribute("value"); attribute != nil && *attribute != "" {
					controlValue = *attribute
				}
				if value == "on" || value == fieldName || value == controlValue {
					if checked, err := element.Eval(`() => this.checked === true`); err == nil && checked.Value.Bool() {
						return true
					}
					if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
						c.logger.Debug("Failed to check input",
							slog.String("field", fieldName),
							slog.String("type", inputType),
							slog.String("error", err.Error()),
						)
					}
				}
			case "hidden", "submit", "button", "reset", "image", "file":
				// Not text-fillable. Hidden values are submitted as-is.
			case "date", "time", "datetime-local", "month", "week":
				// chrome ignores typed text in these pickers, so the value is set directly
				if _, err := element.Eval(`(v) => {
					this.value = v;
					this.dispatchEvent(new Event("input", {bubbles: true}));
					this.dispatchEvent(new Event("change", {bubbles: true}));
				}`, value); err != nil {
					c.logger.Debug("Failed to set input value",
						slog.String("field", fieldName),
						slog.String("type", inputType),
						slog.String("error", err.Error()),
					)
				}
			default:
				if err := element.Input(value); err != nil {
					c.logger.Debug("Failed to fill input field",
						slog.String("field", fieldName),
						slog.String("value", value),
						slog.String("error", err.Error()),
					)
				}
			}

		case "TEXTAREA":
			if err := element.Input(value); err != nil {
				c.logger.Debug("Failed to fill textarea",
					slog.String("field", fieldName),
					slog.String("value", value),
					slog.String("error", err.Error()),
				)
			}

		case "SELECT":
			if err := element.Select([]string{value}, true, rod.SelectorTypeText); err != nil {
				valueSelector := fmt.Sprintf(`[value="%s"]`, value)
				if err := element.Select([]string{valueSelector}, true, rod.SelectorTypeCSSSector); err != nil {
					c.logger.Debug("Failed to select option",
						slog.String("field", fieldName),
						slog.String("value", value),
						slog.String("error", err.Error()),
					)
				}
			}
		}

		return true
	})

	return nil
}
