package internal

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
	"strings"
)

type Prompt interface {
	Select(label string, toSelect []string, searcher func(input string, index int) bool) (index int, value string, err error)
	Prompt(label string, dfault string) (string, error)
}

type Prompter struct{}

func (receiver Prompter) Select(label string, toSelect []string, searcher func(input string, index int) bool) (int, string, error) {
	prompt := promptui.Select{
		Label:             label,
		Items:             toSelect,
		Size:              20,
		Searcher:          searcher,
		StartInSearchMode: searcher != nil,
	}
	index, value, err := prompt.Run()
	if err != nil {
		return 0, "", err
	}
	return index, value, nil
}

func (receiver Prompter) Prompt(label string, dfault string) (string, error) {
	prompt := promptui.Prompt{
		Label:     label,
		Default:   dfault,
		AllowEdit: false,
	}
	val, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return val, nil
}

func fuzzySearchWithPrefixAnchor(itemsToSelect []string, linePrefix string) func(input string, index int) bool {
	return func(input string, index int) bool {
		role := itemsToSelect[index]

		if strings.HasPrefix(input, linePrefix) {
			if strings.HasPrefix(role, input) {
				return true
			}
			return false
		} else {
			if fuzzy.MatchFold(input, role) {
				return true
			}
		}
		return false
	}
}
