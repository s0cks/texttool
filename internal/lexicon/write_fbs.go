package lexicon

import (
	_ "embed"
	"slices"

	flatbuffers "github.com/google/flatbuffers/go"
	schema "github.com/s0cks/texttool/internal/lexicon/schema"
)

func WriteLexiconFbs(lex *Lexicon) ([]byte, error) {
	builder := flatbuffers.NewBuilder(4096)
	var wordOffsets []flatbuffers.UOffsetT
	for key, val := range lex.Words {
		kOffset := builder.CreateString(key)
		vOffset := builder.CreateString(val)

		schema.WordStart(builder)
		schema.WordAddKey(builder, kOffset)
		schema.WordAddValue(builder, vOffset)
		wordOffsets = append(wordOffsets, schema.WordEnd(builder))
	}

	schema.LexiconStartWordsVector(builder, len(wordOffsets))
	for _, wordOffset := range slices.Backward(wordOffsets) {
		builder.PrependUOffsetT(wordOffset)
	}

	wordsVectorOffset := builder.EndVector(len(wordOffsets))
	var handleOffsets []flatbuffers.UOffsetT
	for key, val := range lex.Handles {
		kOffset := builder.CreateString(key)
		vOffset := builder.CreateString(val)

		schema.HandleStart(builder)
		schema.HandleAddKey(builder, kOffset)
		schema.HandleAddValue(builder, vOffset)
		handleOffsets = append(handleOffsets, schema.HandleEnd(builder))
	}

	schema.LexiconStartHandlesVector(builder, len(handleOffsets))
	for _, handleOffset := range slices.Backward(handleOffsets) {
		builder.PrependUOffsetT(handleOffset)
	}

	handlesVectorOffset := builder.EndVector(len(handleOffsets))
	var acronymOffsets []flatbuffers.UOffsetT
	for key, acr := range lex.Acronyms {
		sayStr := builder.CreateString(acr.Say)
		spellStr := builder.CreateString(acr.Spell)
		expandStr := builder.CreateString(acr.Expand)

		schema.AcronymMapStart(builder)
		schema.AcronymMapAddSay(builder, sayStr)
		schema.AcronymMapAddSpell(builder, spellStr)
		schema.AcronymMapAddExpand(builder, expandStr)
		acrMapOffset := schema.AcronymMapEnd(builder)

		kOffset := builder.CreateString(key)

		schema.AcronymStart(builder)
		schema.AcronymAddKey(builder, kOffset)
		schema.AcronymAddValue(builder, acrMapOffset)
		acronymOffsets = append(acronymOffsets, schema.AcronymEnd(builder))
	}

	schema.LexiconStartAcronymsVector(builder, len(acronymOffsets))
	for _, acronymOffset := range slices.Backward(acronymOffsets) {
		builder.PrependUOffsetT(acronymOffset)
	}

	acronymsVectorOffset := builder.EndVector(len(acronymOffsets))
	schema.LexiconStart(builder)
	schema.LexiconAddWords(builder, wordsVectorOffset)
	schema.LexiconAddHandles(builder, handlesVectorOffset)
	schema.LexiconAddAcronyms(builder, acronymsVectorOffset)
	lexiconOffset := schema.LexiconEnd(builder)

	builder.Finish(lexiconOffset)
	return builder.FinishedBytes(), nil
}
