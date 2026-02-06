//internal/falco/decode.go
package falco

import "log"

func DecodeEvent(raw FalcoEventRaw) Event {
	event := Event{
		Time:         raw.Time,
		Rule:         raw.Rule,
		Priority:     raw.Priority,
		Source:       raw.Source,
		Hostname:     raw.Hostname,
		Output:       raw.Output,
		Tags:         raw.Tags,
		OutputFields: raw.OutputFields,
	}

	// minimal sanity check
	if event.Rule == "" {
		log.Println("warning: falco event without rule")
	}

	return event
}