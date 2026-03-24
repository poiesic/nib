package storydb

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mithrandie/csvq-driver"
	"github.com/oklog/ulid/v2"
)

const storydbDir = "storydb"

// DB wraps a csvq-backed *sql.DB for storydb operations.
type DB struct {
	db *sql.DB
}

var (
	entropy   = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	entropyMu sync.Mutex
)

// NewID generates a ULID suitable for use as a record identifier.
func NewID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// tableSchemas defines CSV column headers for each table.
var tableSchemas = map[string][]string{
	"scenes":           {"scene", "pov", "scene_type", "location", "date", "time", "summary", "checksum", "indexed_at"},
	"facts":            {"id", "scene", "category", "summary", "detail", "source_text", "date", "time", "indexed_at"},
	"scene_characters": {"scene", "character", "role", "indexed_at"},
	"locations":        {"id", "name", "type", "description", "first_scene", "indexed_at"},
	"timeline":         {"id", "date", "time", "event", "detail", "scene", "location", "notes"},
}

// Open opens a csvq connection to the storydb directory under projectRoot.
// Creates the directory and CSV files with headers if they don't exist.
func Open(projectRoot string) (*DB, error) {
	dir := filepath.Join(projectRoot, storydbDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating storydb directory: %w", err)
	}

	if err := ensureCSVFiles(dir); err != nil {
		return nil, err
	}

	db, err := sql.Open("csvq", dir)
	if err != nil {
		return nil, fmt.Errorf("opening csvq connection: %w", err)
	}

	// Verify the connection works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging csvq: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// ensureCSVFiles creates CSV files with headers if they don't already exist.
func ensureCSVFiles(dir string) error {
	for name, headers := range tableSchemas {
		path := filepath.Join(dir, name+".csv")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("creating %s.csv: %w", name, err)
		}
		w := csv.NewWriter(f)
		if err := w.Write(headers); err != nil {
			f.Close()
			return fmt.Errorf("writing headers to %s.csv: %w", name, err)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			f.Close()
			return fmt.Errorf("flushing %s.csv: %w", name, err)
		}
		f.Close()
	}
	return nil
}

// --- Scene operations ---

// UpsertScene inserts or replaces a scene record by slug.
func (d *DB) UpsertScene(scene Scene) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM `scenes.csv` WHERE scene = ?", scene.Scene); err != nil {
		return fmt.Errorf("deleting old scene: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO `scenes.csv` (scene, pov, scene_type, location, date, time, summary, checksum, indexed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		scene.Scene, scene.POV, scene.SceneType, scene.Location, scene.Date, scene.Time, scene.Summary, scene.Checksum, scene.IndexedAt,
	); err != nil {
		return fmt.Errorf("inserting scene: %w", err)
	}

	return tx.Commit()
}

// SceneChecksum returns the stored checksum for a scene slug, or empty string if not found.
func (d *DB) SceneChecksum(slug string) (string, error) {
	var checksum string
	err := d.db.QueryRow("SELECT COALESCE(checksum, '') FROM `scenes.csv` WHERE scene = ?", slug).Scan(&checksum)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("querying scene checksum: %w", err)
	}
	return checksum, nil
}

// QueryScenes returns all scene records.
func (d *DB) QueryScenes() ([]Scene, error) {
	rows, err := d.db.Query("SELECT COALESCE(scene,''), COALESCE(pov,''), COALESCE(scene_type,''), COALESCE(location,''), COALESCE(date,''), COALESCE(time,''), COALESCE(summary,''), COALESCE(checksum,''), COALESCE(indexed_at,'') FROM `scenes.csv`")
	if err != nil {
		return nil, fmt.Errorf("querying scenes: %w", err)
	}
	defer rows.Close()

	var scenes []Scene
	for rows.Next() {
		var s Scene
		if err := rows.Scan(&s.Scene, &s.POV, &s.SceneType, &s.Location, &s.Date, &s.Time, &s.Summary, &s.Checksum, &s.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning scene: %w", err)
		}
		scenes = append(scenes, s)
	}
	return scenes, rows.Err()
}

// --- Fact operations ---

// InsertFacts inserts fact records, generating ULIDs for any without an ID.
func (d *DB) InsertFacts(facts []Fact) error {
	if len(facts) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for i := range facts {
		if facts[i].ID == "" {
			facts[i].ID = NewID()
		}
		if _, err := tx.Exec(
			"INSERT INTO `facts.csv` (id, scene, category, summary, detail, source_text, date, time, indexed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			facts[i].ID, facts[i].Scene, facts[i].Category, facts[i].Summary, facts[i].Detail, facts[i].SourceText, facts[i].Date, facts[i].Time, facts[i].IndexedAt,
		); err != nil {
			return fmt.Errorf("inserting fact: %w", err)
		}
	}

	return tx.Commit()
}

// QueryFacts returns all fact records.
func (d *DB) QueryFacts() ([]Fact, error) {
	rows, err := d.db.Query("SELECT COALESCE(id,''), COALESCE(scene,''), COALESCE(category,''), COALESCE(summary,''), COALESCE(detail,''), COALESCE(source_text,''), COALESCE(date,''), COALESCE(time,''), COALESCE(indexed_at,'') FROM `facts.csv`")
	if err != nil {
		return nil, fmt.Errorf("querying facts: %w", err)
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Scene, &f.Category, &f.Summary, &f.Detail, &f.SourceText, &f.Date, &f.Time, &f.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning fact: %w", err)
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// --- SceneCharacter operations ---

// InsertSceneCharacters inserts scene character records.
func (d *DB) InsertSceneCharacters(chars []SceneCharacter) error {
	if len(chars) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for _, ch := range chars {
		if _, err := tx.Exec(
			"INSERT INTO `scene_characters.csv` (scene, character, role, indexed_at) VALUES (?, ?, ?, ?)",
			ch.Scene, ch.Character, ch.Role, ch.IndexedAt,
		); err != nil {
			return fmt.Errorf("inserting scene character: %w", err)
		}
	}

	return tx.Commit()
}

// QuerySceneCharacters returns all scene character records.
func (d *DB) QuerySceneCharacters() ([]SceneCharacter, error) {
	rows, err := d.db.Query("SELECT COALESCE(scene,''), COALESCE(character,''), COALESCE(role,''), COALESCE(indexed_at,'') FROM `scene_characters.csv`")
	if err != nil {
		return nil, fmt.Errorf("querying scene characters: %w", err)
	}
	defer rows.Close()

	var chars []SceneCharacter
	for rows.Next() {
		var sc SceneCharacter
		if err := rows.Scan(&sc.Scene, &sc.Character, &sc.Role, &sc.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning scene character: %w", err)
		}
		chars = append(chars, sc)
	}
	return chars, rows.Err()
}

// QueryDistinctCharacters returns unique character slugs, sorted ascending.
// When roles is non-empty, only characters with a matching role are included.
func (d *DB) QueryDistinctCharacters(roles []string) ([]string, error) {
	query := "SELECT DISTINCT character FROM `scene_characters.csv` WHERE character != ''"
	var args []any
	if len(roles) > 0 {
		placeholders := make([]string, len(roles))
		for i, r := range roles {
			placeholders[i] = "?"
			args = append(args, r)
		}
		query += " AND role IN (" + strings.Join(placeholders, ",") + ")"
	}
	query += " ORDER BY character"
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying distinct characters: %w", err)
	}
	defer rows.Close()

	var characters []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning character: %w", err)
		}
		characters = append(characters, name)
	}
	return characters, rows.Err()
}

// QueryDistinctCharactersBySlugs returns unique character slugs from the given
// scene slugs, filtered by roles, sorted ascending.
func (d *DB) QueryDistinctCharactersBySlugs(slugs, roles []string) ([]string, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	slugPH := make([]string, len(slugs))
	args := make([]any, 0, len(slugs)+len(roles))
	for i, s := range slugs {
		slugPH[i] = "?"
		args = append(args, s)
	}
	query := "SELECT DISTINCT character FROM `scene_characters.csv` WHERE character != '' AND scene IN (" + strings.Join(slugPH, ",") + ")"
	if len(roles) > 0 {
		rolePH := make([]string, len(roles))
		for i, r := range roles {
			rolePH[i] = "?"
			args = append(args, r)
		}
		query += " AND role IN (" + strings.Join(rolePH, ",") + ")"
	}
	query += " ORDER BY character"
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying distinct characters by slugs: %w", err)
	}
	defer rows.Close()

	var characters []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning character: %w", err)
		}
		characters = append(characters, name)
	}
	return characters, rows.Err()
}

// --- Location operations ---

// InsertLocations inserts location records, generating ULIDs for any without an ID.
func (d *DB) InsertLocations(locs []Location) error {
	if len(locs) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for i := range locs {
		if locs[i].ID == "" {
			locs[i].ID = NewID()
		}
		if _, err := tx.Exec(
			"INSERT INTO `locations.csv` (id, name, type, description, first_scene, indexed_at) VALUES (?, ?, ?, ?, ?, ?)",
			locs[i].ID, locs[i].Name, locs[i].Type, locs[i].Description, locs[i].FirstScene, locs[i].IndexedAt,
		); err != nil {
			return fmt.Errorf("inserting location: %w", err)
		}
	}

	return tx.Commit()
}

// QueryLocations returns all location records.
func (d *DB) QueryLocations() ([]Location, error) {
	rows, err := d.db.Query("SELECT COALESCE(id,''), COALESCE(name,''), COALESCE(type,''), COALESCE(description,''), COALESCE(first_scene,''), COALESCE(indexed_at,'') FROM `locations.csv`")
	if err != nil {
		return nil, fmt.Errorf("querying locations: %w", err)
	}
	defer rows.Close()

	var locs []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Type, &l.Description, &l.FirstScene, &l.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning location: %w", err)
		}
		locs = append(locs, l)
	}
	return locs, rows.Err()
}

// --- Filtered query operations ---

// QueryScenesBySlugs returns scene records matching any of the given slugs.
func (d *DB) QueryScenesBySlugs(slugs []string) ([]Scene, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(slugs))
	args := make([]any, len(slugs))
	for i, s := range slugs {
		placeholders[i] = "?"
		args[i] = s
	}
	query := fmt.Sprintf(
		"SELECT COALESCE(scene,''), COALESCE(pov,''), COALESCE(scene_type,''), COALESCE(location,''), COALESCE(date,''), COALESCE(time,''), COALESCE(summary,''), COALESCE(checksum,''), COALESCE(indexed_at,'') FROM `scenes.csv` WHERE scene IN (%s)",
		strings.Join(placeholders, ","),
	)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying scenes by slugs: %w", err)
	}
	defer rows.Close()

	var scenes []Scene
	for rows.Next() {
		var s Scene
		if err := rows.Scan(&s.Scene, &s.POV, &s.SceneType, &s.Location, &s.Date, &s.Time, &s.Summary, &s.Checksum, &s.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning scene: %w", err)
		}
		scenes = append(scenes, s)
	}
	return scenes, rows.Err()
}

// QueryFactsBySlugs returns fact records for scenes matching any of the given slugs.
func (d *DB) QueryFactsBySlugs(slugs []string) ([]Fact, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(slugs))
	args := make([]any, len(slugs))
	for i, s := range slugs {
		placeholders[i] = "?"
		args[i] = s
	}
	query := fmt.Sprintf(
		"SELECT COALESCE(id,''), COALESCE(scene,''), COALESCE(category,''), COALESCE(summary,''), COALESCE(detail,''), COALESCE(source_text,''), COALESCE(date,''), COALESCE(time,''), COALESCE(indexed_at,'') FROM `facts.csv` WHERE scene IN (%s)",
		strings.Join(placeholders, ","),
	)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying facts by slugs: %w", err)
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Scene, &f.Category, &f.Summary, &f.Detail, &f.SourceText, &f.Date, &f.Time, &f.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning fact: %w", err)
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// QueryCharactersBySlugs returns scene character records for scenes matching any of the given slugs.
func (d *DB) QueryCharactersBySlugs(slugs []string) ([]SceneCharacter, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(slugs))
	args := make([]any, len(slugs))
	for i, s := range slugs {
		placeholders[i] = "?"
		args[i] = s
	}
	query := fmt.Sprintf(
		"SELECT COALESCE(scene,''), COALESCE(character,''), COALESCE(role,''), COALESCE(indexed_at,'') FROM `scene_characters.csv` WHERE scene IN (%s)",
		strings.Join(placeholders, ","),
	)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying characters by slugs: %w", err)
	}
	defer rows.Close()

	var chars []SceneCharacter
	for rows.Next() {
		var sc SceneCharacter
		if err := rows.Scan(&sc.Scene, &sc.Character, &sc.Role, &sc.IndexedAt); err != nil {
			return nil, fmt.Errorf("scanning scene character: %w", err)
		}
		chars = append(chars, sc)
	}
	return chars, rows.Err()
}

// QuerySceneSlugsForCharacters returns scene slugs where any of the given characters appear.
func (d *DB) QuerySceneSlugsForCharacters(characters []string) ([]string, error) {
	if len(characters) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(characters))
	args := make([]any, len(characters))
	for i, c := range characters {
		placeholders[i] = "?"
		args[i] = c
	}
	query := fmt.Sprintf(
		"SELECT DISTINCT scene FROM `scene_characters.csv` WHERE character IN (%s)",
		strings.Join(placeholders, ","),
	)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying scene slugs for characters: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("scanning scene slug: %w", err)
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

// QuerySceneSlugsForCharactersWithRoles returns scene slugs where any of the
// given characters appear with one of the specified roles.
func (d *DB) QuerySceneSlugsForCharactersWithRoles(characters, roles []string) ([]string, error) {
	if len(characters) == 0 || len(roles) == 0 {
		return nil, nil
	}
	charPH := make([]string, len(characters))
	args := make([]any, 0, len(characters)+len(roles))
	for i, c := range characters {
		charPH[i] = "?"
		args = append(args, c)
	}
	rolePH := make([]string, len(roles))
	for i, r := range roles {
		rolePH[i] = "?"
		args = append(args, r)
	}
	query := fmt.Sprintf(
		"SELECT DISTINCT scene FROM `scene_characters.csv` WHERE character IN (%s) AND role IN (%s)",
		strings.Join(charPH, ","), strings.Join(rolePH, ","),
	)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying scene slugs for characters with roles: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("scanning scene slug: %w", err)
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

// --- Generic operations ---

// DeleteByScene removes all records matching a scene slug from the specified table.
func (d *DB) DeleteByScene(table, slug string) error {
	query := fmt.Sprintf("DELETE FROM `%s.csv` WHERE scene = ?", table)
	if _, err := d.db.Exec(query, slug); err != nil {
		return fmt.Errorf("deleting from %s where scene=%q: %w", table, slug, err)
	}
	return nil
}

// Reset deletes all records from every storydb table, leaving headers intact.
func (d *DB) Reset() error {
	for name := range tableSchemas {
		query := fmt.Sprintf("DELETE FROM `%s.csv`", name)
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("clearing %s: %w", name, err)
		}
	}
	return nil
}

// RenameScene updates the scene column from oldSlug to newSlug across
// scenes, facts, and scene_characters tables.
func (d *DB) RenameScene(oldSlug, newSlug string) error {
	for _, table := range []string{"scenes", "facts", "scene_characters"} {
		query := fmt.Sprintf("UPDATE `%s.csv` SET scene = ? WHERE scene = ?", table)
		if _, err := d.db.Exec(query, newSlug, oldSlug); err != nil {
			return fmt.Errorf("renaming scene in %s: %w", table, err)
		}
	}
	return nil
}
