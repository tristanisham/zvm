// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/schollz/progressbar/v3"
)

func TestStopProgressBar(t *testing.T) {
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
			pbar := progressbar.NewOptions64(tt.contentLength,
				progressbar.OptionSetWriter(io.Discard),
				progressbar.OptionShowBytes(true),
				progressbar.OptionShowCount(),
				progressbar.OptionSetRenderBlankState(true),
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
			stopProgressBar(pbar, err == nil)

			// The background spinner exits only when IsFinished becomes true.
			wantFinished := !tt.fail || tt.contentLength == -1
			if got := pbar.IsFinished(); got != wantFinished {
				t.Errorf("IsFinished() = %v, want %v", got, wantFinished)
			}
			if got := pbar.State().CurrentBytes; got != 3 {
				t.Errorf("downloaded bytes = %v, want 3", got)
			}
		})
	}
}
