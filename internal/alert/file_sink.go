//internal/alert/file_sink.go
package alert

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type FileSink struct {
	mu   sync.Mutex
	file *os.File
}

func NewFileSink(path string) (*FileSink, error) {
	f, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, err
	}

	return &FileSink{file: f}, nil
}

func (s *FileSink) Emit(
	_ context.Context,
	alert *model.Alert,
) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	_, err = s.file.Write(append(data, '\n'))
	return err
}