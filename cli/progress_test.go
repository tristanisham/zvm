// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/schollz/progressbar/v3"
)

// writeCounter counts writes; the spinner ticker writes from its own goroutine.
type writeCounter struct{ n atomic.Int64 }

func (w *writeCounter) Write(p []byte) (int, error) {
	w.n.Add(1)
	return len(p), nil
}

func TestStopProgressBar(t *testing.T) {
	const spinnerInterval = 10 * time.Millisecond

	for _, tt := range []struct {
		name          string
		contentLength int64
		fail          bool
	}{
		{"known length success", 3, false},
		{"known length failure", 10, true},
		{"unknown length success", -1, false},
		{"unknown length failure", -1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &writeCounter{}
			pbar := progressbar.NewOptions64(tt.contentLength,
				progressbar.OptionSetWriter(out),
				progressbar.OptionShowBytes(true),
				progressbar.OptionShowCount(),
				progressbar.OptionSetRenderBlankState(true),
				progressbar.OptionSetSpinnerChangeInterval(spinnerInterval),
			)
			t.Cleanup(func() {
				// Also stop the ticker if an assertion fails after Exit.
				pbar.Reset()
				_ = pbar.Finish()
			})

			var source io.Reader = strings.NewReader("abc")
			readErr := errors.New("download interrupted")
			if tt.fail {
				source = io.MultiReader(source, iotest.ErrReader(readErr))
			}
			_, err := io.Copy(pbar, source)
			if tt.fail && !errors.Is(err, readErr) || !tt.fail && err != nil {
				t.Fatalf("unexpected copy error: %v", err)
			}

			// Let an unknown-length spinner redraw at least once before stopping,
			// so the check below would fail if stopProgressBar did nothing.
			time.Sleep(5 * spinnerInterval)
			stopProgressBar(pbar, err == nil)

			// Nothing may be drawn after the bar is stopped.
			before := out.n.Load()
			time.Sleep(5 * spinnerInterval)
			if after := out.n.Load(); after != before {
				t.Errorf("progress bar wrote %d times after being stopped", after-before)
			}
			if got := pbar.State().CurrentBytes; got != 3 {
				t.Errorf("downloaded bytes = %v, want 3", got)
			}
		})
	}
}
