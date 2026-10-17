// Package emit writes documents as JSON or NDJSON.
package emit

import (
	"bufio"
	"encoding/json"
	"io"

	"github.com/s0cks/texttool/internal/doc"
)

// Write encodes docs to w. In JSON mode a single document is written as an
// object and several as an array. In NDJSON mode every segment and chunk is
// one line, tagged with its source file when more than one input was given.
func Write(w io.Writer, docs []*doc.Document, ndjson, pretty bool) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	if !ndjson {
		if pretty {
			enc.SetIndent("", "  ")
		}
		if len(docs) == 1 {
			return enc.Encode(docs[0])
		}
		return enc.Encode(docs)
	}
	multi := len(docs) > 1
	for _, d := range docs {
		for i := range d.Segments {
			if multi {
				d.Segments[i].Source = d.Source
			}
			if err := enc.Encode(&d.Segments[i]); err != nil {
				return err
			}
		}
		for i := range d.Chunks {
			if multi {
				d.Chunks[i].Source = d.Source
			}
			if err := enc.Encode(&d.Chunks[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
