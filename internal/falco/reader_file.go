//internal/falco/reader_file.go
package falco

import (
	"bufio"
	"encoding/json"
	"os"
)

func ReadFromFile(path string, out chan<- FalcoEventRaw) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Bytes()

		var event FalcoEventRaw
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		out <- event
	}

	return scanner.Err()
}