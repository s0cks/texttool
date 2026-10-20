package lexicon

import (
	schema "github.com/s0cks/texttool/internal/lexicon/schema"
)

func parseFbs(data []byte) (*Lexicon, error) {
	lex := &Lexicon{
		Acronyms: make(map[string]Acronym),
		Words:    make(map[string]string),
		Handles:  make(map[string]string),
	}

	schemaLexicon := schema.GetRootAsLexicon(data, 0)
	wordsLength := schemaLexicon.WordsLength()
	for i := range wordsLength {
		var wordObj schema.Word
		if schemaLexicon.Words(&wordObj, i) {
			key := string(wordObj.Key())
			val := string(wordObj.Value())
			lex.Words[key] = val
		}
	}

	handlesLength := schemaLexicon.HandlesLength()
	for i := range handlesLength {
		var handleObj schema.Handle
		if schemaLexicon.Handles(&handleObj, i) {
			key := string(handleObj.Key())
			val := string(handleObj.Value())
			lex.Handles[key] = val
		}
	}

	acronymsLength := schemaLexicon.AcronymsLength()
	for i := range acronymsLength {
		var acronymObj schema.Acronym
		if schemaLexicon.Acronyms(&acronymObj, i) {
			key := string(acronymObj.Key())

			var mapObj schema.AcronymMap
			var acr Acronym
			if acronymObj.Value(&mapObj) != nil {
				acr.Say = string(mapObj.Say())
				acr.Spell = string(mapObj.Spell())
				acr.Expand = string(mapObj.Expand())
			}

			lex.Acronyms[key] = acr
		}
	}

	return lex, nil
}
