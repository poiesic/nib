package storydb

// Scene holds metadata about an indexed scene.
type Scene struct {
	Scene     string `json:"scene"`
	POV       string `json:"pov"`
	SceneType string `json:"scene_type"`
	Location  string `json:"location"`
	Date      string `json:"date"`
	Time      string `json:"time"`
	Summary   string `json:"summary"`
	Checksum  string `json:"checksum"`
	IndexedAt string `json:"indexed_at"`
}

// Fact holds a single extracted fact from a scene.
type Fact struct {
	ID         string `json:"id"`
	Scene      string `json:"scene"`
	Category   string `json:"category"`
	Summary    string `json:"summary"`
	Detail     string `json:"detail"`
	SourceText string `json:"source_text"`
	Date       string `json:"date"`
	Time       string `json:"time"`
	IndexedAt  string `json:"indexed_at"`
}

// SceneCharacter records a character's presence in a scene.
type SceneCharacter struct {
	Scene     string `json:"scene"`
	Character string `json:"character"`
	Role      string `json:"role"`
	IndexedAt string `json:"indexed_at"`
}

// Location describes a setting used in the manuscript.
type Location struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	FirstScene  string `json:"first_scene"`
	IndexedAt   string `json:"indexed_at"`
}

// TimelineEntry records a narrative event in chronological order.
type TimelineEntry struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Event    string `json:"event"`
	Detail   string `json:"detail"`
	Scene    string `json:"scene"`
	Location string `json:"location"`
	Notes    string `json:"notes"`
}
