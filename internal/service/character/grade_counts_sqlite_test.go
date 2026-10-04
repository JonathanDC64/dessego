//go:build cgo
// +build cgo

package character

import (
	"database/sql"
	"os"
	"sync"
	"testing"
)

func TestSQLiteGradeCountsAreAtomicAndPreserveOtherStats(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ddl, err := os.ReadFile("character.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(ddl)); err != nil {
		t.Fatal(err)
	}
	s := &SQLiteService{db: db}
	id := "player' OR 1=1 --"
	for _, p := range []string{id, "other"} {
		if _, err = db.Exec("INSERT INTO character(id,sessions,msg_rating) VALUES(?,7,9)", p); err != nil {
			t.Fatal(err)
		}
	}
	read := func(player string) [7]int {
		t.Helper()
		var got [7]int
		if err := db.QueryRow("SELECT grade_s,grade_a,grade_b,grade_c,grade_d,sessions,msg_rating FROM character WHERE id=?", player).Scan(&got[0], &got[1], &got[2], &got[3], &got[4], &got[5], &got[6]); err != nil {
			t.Fatal(err)
		}
		return got
	}
	for _, g := range []GradeCounts{{S: 2}, {S: 1, A: 1}, {B: 1, C: 1, D: 2}, {}} {
		if err := s.UpdatePlayerGrades(id, g); err != nil {
			t.Fatal(err)
		}
	}
	want := [7]int{3, 1, 1, 1, 2, 7, 9}
	if got := read(id); got != want {
		t.Fatalf("counts=%v want=%v", got, want)
	}
	for _, g := range []GradeCounts{{S: 1, A: -1}, {S: 1, D: MaxGradeCount + 1}, {S: 1, D: MaxGradeCount}} {
		if err := s.UpdatePlayerGrades(id, g); err == nil {
			t.Fatalf("accepted %+v", g)
		}
		if got := read(id); got != want {
			t.Fatalf("failed update partially changed counts: %v", got)
		}
	}
	if err := s.UpdatePlayerGrades("missing", GradeCounts{S: 2}); err == nil {
		t.Fatal("acknowledged missing character")
	}
	// Concurrent completion reports must add, rather than overwrite, their counts.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.UpdatePlayerGrades(id, GradeCounts{S: 2, A: 1}) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	want = [7]int{19, 9, 1, 1, 2, 7, 9}
	if got := read(id); got != want {
		t.Fatalf("concurrent counts=%v want=%v", got, want)
	}
	if got := read("other"); got != ([7]int{0, 0, 0, 0, 0, 7, 9}) {
		t.Fatalf("changed other character: %v", got)
	}
	// Reaching the largest encodable total succeeds; exceeding it changes nothing.
	if err := s.UpdatePlayerGrades(id, GradeCounts{D: MaxGradeCount - 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePlayerGrades(id, GradeCounts{S: 1, D: 1}); err == nil {
		t.Fatal("accepted overflowing total")
	}
	want[4] = MaxGradeCount
	if got := read(id); got != want {
		t.Fatalf("overflow partially changed counts: %v", got)
	}
}
