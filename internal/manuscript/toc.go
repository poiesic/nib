package manuscript

import (
	"fmt"
	"io"
	"os"

	"github.com/poiesic/nib/internal/bookio"
)

// TOC prints a table of contents showing chapters and scenes in dotted notation.
// Output is tab-separated for easy processing with cut, awk, etc.
func TOC(w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}

	_, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	chapterCount := 0
	for i, ch := range book.Chapters {
		chNum := i + 1
		if !ch.Interlude {
			chapterCount++
		}

		heading := chapterHeading(ch.Name, ch.Interlude, chapterCount)
		fmt.Fprintf(w, "%d\t%s\n", chNum, heading)

		for j, slug := range ch.Scenes {
			fmt.Fprintf(w, "%d.%d\t%s\n", chNum, j+1, slug)
		}
	}

	return nil
}

func chapterHeading(name string, interlude bool, num int) string {
	if name != "" {
		return name
	}
	if interlude {
		return "Interlude"
	}
	return fmt.Sprintf("Chapter %d", num)
}
