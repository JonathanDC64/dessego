package game

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danmrichards/dessego/internal/service/character"
	"github.com/rs/zerolog"
)

type completionCharacters struct {
	Characters
	id     string
	counts character.GradeCounts
	calls  int
	err    error
}

func (c *completionCharacters) UpdatePlayerGrades(id string, counts character.GradeCounts) error {
	c.id, c.counts = id, counts
	c.calls++
	return c.err
}

func TestFinaliseMultiplayerRatingCounts(t *testing.T) {
	for _, tc := range []struct {
		name, form string
		want       character.GradeCounts
	}{
		// Anonymized from the three-player completion captured on 2026-10-04 UTC.
		{"captured three-player completion", "presence=67108864&gradeS=2&gradeA=0&gradeB=0&gradeC=0&gradeD=0", character.GradeCounts{S: 2}},
		{"two-player completion", "gradeS=1", character.GradeCounts{S: 1}},
		{"mixed ratings", "gradeS=1&gradeA=1", character.GradeCounts{S: 1, A: 1}},
		{"other buckets", "gradeB=1&gradeC=1&gradeD=2", character.GradeCounts{B: 1, C: 1, D: 2}},
		{"unrated completion", "gradeS=0&gradeA=0&gradeB=0&gradeC=0&gradeD=0", character.GradeCounts{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cs := &completionCharacters{}
			s := &Server{cs: cs, rd: sosPlaintextDecrypter{}, l: zerolog.Nop()}
			w := httptest.NewRecorder()
			form := "characterID=capture-player0&" + tc.form + "&ver=100&\x00"
			s.finaliseMultiplayHandler()(w, httptest.NewRequest(http.MethodPost, "/cgi-bin/finalizeMultiPlay.spd", strings.NewReader(form)))
			if w.Code != http.StatusOK {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if cs.calls != 1 || cs.id != "capture-player0" || cs.counts != tc.want {
				t.Fatalf("update: calls=%d id=%q counts=%+v, want %+v", cs.calls, cs.id, cs.counts, tc.want)
			}
			data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(w.Body.String()))
			if err != nil {
				t.Fatal(err)
			}
			if len(data) != 6 || data[0] != 0x21 || binary.LittleEndian.Uint32(data[1:5]) != 6 || data[5] != 1 || !strings.HasSuffix(w.Body.String(), "\n") {
				t.Fatalf("invalid completion response: %x", data)
			}
		})
	}
}

func TestFinaliseMultiplayerRejectsInvalidCounts(t *testing.T) {
	for _, form := range []string{"gradeS=-1", "gradeA=-1", "gradeD=2147483648"} {
		cs := &completionCharacters{}
		s := &Server{cs: cs, rd: sosPlaintextDecrypter{}, l: zerolog.Nop()}
		w := httptest.NewRecorder()
		s.finaliseMultiplayHandler()(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("characterID=capture-player0&"+form)))
		if w.Code != http.StatusBadRequest || cs.calls != 0 {
			t.Fatalf("%s: status=%d calls=%d", form, w.Code, cs.calls)
		}
	}
}

func TestFinaliseMultiplayerDoesNotAcknowledgeFailedWrite(t *testing.T) {
	cs := &completionCharacters{err: errors.New("database write failed")}
	s := &Server{cs: cs, rd: sosPlaintextDecrypter{}, l: zerolog.Nop()}
	w := httptest.NewRecorder()
	s.finaliseMultiplayHandler()(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("characterID=capture-player0&gradeS=2")))
	if w.Code != http.StatusInternalServerError || cs.calls != 1 {
		t.Fatalf("status=%d calls=%d", w.Code, cs.calls)
	}
}
