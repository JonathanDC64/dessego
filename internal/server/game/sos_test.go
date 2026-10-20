package game

import (
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danmrichards/dessego/internal/service/sos"
	"github.com/rs/zerolog"
)

type sosPlaintextDecrypter struct{}

func (sosPlaintextDecrypter) Decrypt(data []byte) ([]byte, error) { return data, nil }

func TestGetSOSDiscoversAndRetainsSigns(t *testing.T) {
	m := sos.NewManager(zerolog.Nop())
	m.Add(&sos.SOS{CharacterID: "helper", BlockID: 60070})
	// Add normally stamps Updated through ToSos; keep this fixture alive.
	m.Check("helper")
	s := &Server{sos: m, rd: sosPlaintextDecrypter{}, l: zerolog.Nop()}
	for _, tc := range []struct {
		name, form     string
		known, unknown uint32
	}{
		{"new sign", "blockID=60070&maxSosNum=10&sosNum=0", 0, 1},
		{"known sign", "blockID=60070&maxSosNum=10&sosNum=1&sosList=1", 1, 0},
		{"different block", "blockID=60071&maxSosNum=10&sosNum=0", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/cgi-bin/getSosData.spd", strings.NewReader(tc.form))
			w := httptest.NewRecorder()
			s.getSosDataHandler()(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(w.Body.String()))
			if err != nil {
				t.Fatal(err)
			}
			if len(data) < 13 || data[0] != 0x0f || binary.LittleEndian.Uint32(data[1:5]) != uint32(len(data)) {
				t.Fatal("invalid SOS response framing")
			}
			known := binary.LittleEndian.Uint32(data[5:9])
			if known != tc.known {
				t.Fatalf("known signs = %d, want %d", known, tc.known)
			}
			unknown := binary.LittleEndian.Uint32(data[9+4*known : 13+4*known])
			if unknown != tc.unknown {
				t.Fatalf("new signs = %d, want %d", unknown, tc.unknown)
			}
			if known == 1 && binary.LittleEndian.Uint32(data[9:13]) != 1 {
				t.Fatal("wrong known sign ID")
			}
			if unknown == 1 && binary.LittleEndian.Uint32(data[13:17]) != 1 {
				t.Fatal("wrong new sign ID")
			}
		})
	}
}

func TestGetSOSMatchesGroupHeader(t *testing.T) {
	m := sos.NewManager(zerolog.Nop())
	m.Add(&sos.SOS{CharacterID: "public", BlockID: 60070})
	m.Add(&sos.SOS{CharacterID: "friend", BlockID: 60070, Group: "ab12"})
	m.Check("public")
	m.Check("friend")
	s := &Server{sos: m, rd: sosPlaintextDecrypter{}, l: zerolog.Nop()}
	for _, tc := range []struct {
		name, header string
		want         uint32
	}{
		{"no header", "", 1},
		{"the group", "group=ab12", 2},
		{"another group", "tendency=local; group=cd34", 3},
		{"malformed is public", "group=AB12", 1},
		{"not an option", "ab12", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/cgi-bin/getSosData.spd",
				strings.NewReader("blockID=60070&maxSosNum=10&sosNum=0"))
			if tc.header != "" {
				r.Header.Set(clientOptionsHeader, tc.header)
			}
			w := httptest.NewRecorder()
			s.getSosDataHandler()(w, r)
			data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(w.Body.String()))
			if err != nil || len(data) < 13 {
				t.Fatalf("bad response %q", w.Body.String())
			}
			known := binary.LittleEndian.Uint32(data[5:9])
			unknown := binary.LittleEndian.Uint32(data[9+4*known : 13+4*known])
			ids := map[uint32]bool{}
			if unknown == 1 {
				ids[binary.LittleEndian.Uint32(data[13+4*known:17+4*known])] = true
			}
			switch tc.want {
			case 1: // the public sign (ID 1) only
				if unknown != 1 || !ids[1] {
					t.Fatalf("signs = %d %v, want the public one", unknown, ids)
				}
			case 2: // the group's sign (ID 2) only
				if unknown != 1 || !ids[2] {
					t.Fatalf("signs = %d %v, want the group's", unknown, ids)
				}
			case 3: // none
				if unknown != 0 {
					t.Fatalf("signs = %d, want none", unknown)
				}
			}
		})
	}
}
