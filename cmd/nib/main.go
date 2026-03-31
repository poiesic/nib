package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/chapter"
	"github.com/poiesic/nib/internal/character"
	"github.com/poiesic/nib/internal/continuity"
	"github.com/poiesic/nib/internal/manuscript"
	"github.com/poiesic/nib/internal/project"
	"github.com/poiesic/nib/internal/project/templates"
	"github.com/poiesic/nib/internal/scene"
	"github.com/poiesic/nib/internal/version"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:                  "nib",
		Usage:                 "a novel-writing CLI tool",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "pretty",
				Usage: "pretty-print JSON output",
			},
		},
		Commands: []*cli.Command{
			initCommand(),
			chapterCommand(),
			profileCommand(),
			sceneCommand(),
			manuscriptCommand(),
			continuityCommand(),
			{
				Name:  "styles",
				Usage: "list available STYLE.md variants for nib init",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					for _, s := range templates.ValidStyles {
						fmt.Println(s)
					}
					return nil
				},
			},
			{
				Name:  "version",
				Usage: "show version and build information",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println(version.String())
					return nil
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func initCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "scaffold a new novel project",
		ArgsUsage: "<project-name>",
		Aliases:   []string{"in"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-git",
				Usage: "skip git repo initialization",
			},
			&cli.StringFlag{
				Name:  "style",
				Usage: fmt.Sprintf("STYLE.md variant: %s", strings.Join(templates.ValidStyles, ", ")),
				Value: "first-person",
			},
			&cli.BoolFlag{
				Name:  "no-style",
				Usage: "skip STYLE.md creation",
			},
			&cli.StringFlag{
				Name:  "agent",
				Usage: "AI agent backend for project scaffolding",
				Value: "claude",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() != 1 {
				return fmt.Errorf("usage: nib init <project-name>")
			}
			opts := project.InitOptions{
				NoGit:   cmd.Bool("no-git"),
				Style:   cmd.String("style"),
				NoStyle: cmd.Bool("no-style"),
				Agent:   cmd.String("agent"),
			}
			resolved, err := project.Init(args.First(), opts)
			if err != nil {
				return err
			}
			fmt.Printf("Initialized project %s/\n", resolved)
			return nil
		},
	}
}

func chapterCommand() *cli.Command {
	return &cli.Command{
		Name:    "chapter",
		Aliases: []string{"ch"},
		Usage:   "manage chapters in book.yaml",
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"ad"},
				Usage:   "add a new chapter",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "name",
						Usage: "chapter name (omit for auto-numbered)",
					},
					&cli.BoolFlag{
						Name:  "interlude",
						Usage: "mark as interlude",
					},
					&cli.IntFlag{
						Name:  "at",
						Usage: "1-based position to insert at (0 or omit to append)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					opts := chapter.AddOptions{
						Name:      cmd.String("name"),
						Interlude: cmd.Bool("interlude"),
						At:        int(cmd.Int("at")),
					}
					if err := chapter.Add(opts); err != nil {
						return err
					}
					if opts.Name != "" {
						fmt.Printf("Added chapter %q\n", opts.Name)
					} else if opts.Interlude {
						fmt.Println("Added interlude")
					} else {
						fmt.Println("Added chapter")
					}
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"li"},
				Usage:   "list all chapters",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					chapters, err := chapter.List()
					if err != nil {
						return err
					}
					fmt.Print(chapter.FormatList(chapters))
					return nil
				},
			},
			{
				Name:      "name",
				Aliases:   []string{"na"},
				Usage:     "set a chapter name",
				ArgsUsage: "<index> <name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 2 {
						return fmt.Errorf("usage: nib chapter name <index> <name>")
					}
					index, err := strconv.Atoi(args.Get(0))
					if err != nil {
						return fmt.Errorf("invalid index: %s", args.Get(0))
					}
					if err := chapter.Name(index, args.Get(1)); err != nil {
						return err
					}
					fmt.Printf("Named chapter %d %q\n", index, args.Get(1))
					return nil
				},
			},
			{
				Name:      "clear-name",
				Aliases:   []string{"cn"},
				Usage:     "remove a chapter name",
				ArgsUsage: "<index>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib chapter clear-name <index>")
					}
					index, err := strconv.Atoi(args.First())
					if err != nil {
						return fmt.Errorf("invalid index: %s", args.First())
					}
					if err := chapter.ClearName(index); err != nil {
						return err
					}
					fmt.Printf("Cleared name from chapter %d\n", index)
					return nil
				},
			},
			{
				Name:      "move",
				Aliases:   []string{"mo"},
				Usage:     "move a chapter to a new position",
				ArgsUsage: "<from> <to>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 2 {
						return fmt.Errorf("usage: nib chapter move <from> <to>")
					}
					from, err := strconv.Atoi(args.Get(0))
					if err != nil {
						return fmt.Errorf("invalid source index: %s", args.Get(0))
					}
					to, err := strconv.Atoi(args.Get(1))
					if err != nil {
						return fmt.Errorf("invalid destination index: %s", args.Get(1))
					}
					if err := chapter.Move(from, to); err != nil {
						return err
					}
					fmt.Printf("Moved chapter %d to position %d\n", from, to)
					return nil
				},
			},
			{
				Name:      "remove",
				Aliases:   []string{"rm"},
				Usage:     "remove a chapter by index",
				ArgsUsage: "<index>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib chapter remove <index>")
					}
					index, err := strconv.Atoi(args.First())
					if err != nil {
						return fmt.Errorf("invalid index: %s", args.First())
					}
					if err := chapter.Remove(index); err != nil {
						return err
					}
					fmt.Printf("Removed chapter %d\n", index)
					return nil
				},
			},
		},
	}
}

func profileCommand() *cli.Command {
	return &cli.Command{
		Name:    "profile",
		Aliases: []string{"pr"},
		Usage:   "manage character profiles",
		Commands: []*cli.Command{
			{
				Name:      "add",
				Aliases:   []string{"ad"},
				Usage:     "create a new character profile",
				ArgsUsage: "<slug>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib profile add <slug>")
					}
					path, err := character.Add(args.First())
					if err != nil {
						return err
					}
					fmt.Printf("Created %s\n", path)
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"li"},
				Usage:   "list all character profiles",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					chars, err := character.List()
					if err != nil {
						return err
					}
					fmt.Print(character.FormatList(chars))
					return nil
				},
			},
			{
				Name:      "edit",
				Aliases:   []string{"ed"},
				Usage:     "open a character profile in your editor",
				ArgsUsage: "<slug>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib profile edit <slug>")
					}
					return character.Edit(character.EditOptions{Slug: args.First()})
				},
			},
			{
				Name:      "talk",
				Aliases:   []string{"ta"},
				Usage:     "role-play as a character at a specific point in the story",
				ArgsUsage: "<slug> <scene>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "resume",
						Usage: "resume an existing talk session",
					},
					&cli.BoolFlag{
						Name:  "new",
						Usage: "delete existing session and start fresh",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 2 {
						return fmt.Errorf("usage: nib profile talk [--resume|--new] <slug> <scene>\n\nExamples:\n  nib pr talk lance-thurgood 37.2\n  nib pr ta --resume lance-thurgood 37.2\n  nib pr ta --new lance-thurgood 37.2")
					}
					return character.Talk(character.TalkOptions{
						Slug:   args.Get(0),
						Scene:  args.Get(1),
						Resume: cmd.Bool("resume"),
						New:    cmd.Bool("new"),
					})
				},
			},
			{
				Name:      "remove",
				Aliases:   []string{"rm"},
				Usage:     "remove a character profile",
				ArgsUsage: "<slug>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib profile remove <slug>")
					}
					slug := args.First()
					if err := character.Remove(slug); err != nil {
						return err
					}
					fmt.Printf("Removed character %q\n", slug)
					return nil
				},
			},
		},
	}
}

func sceneCommand() *cli.Command {
	return &cli.Command{
		Name:    "scene",
		Aliases: []string{"sc"},
		Usage:   "manage scenes within chapters",
		Commands: []*cli.Command{
			{
				Name:      "add",
				Aliases:   []string{"ad"},
				Usage:     "add a scene to a chapter",
				ArgsUsage: "<chapter-index> <slug>",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "at",
						Usage: "1-based position to insert at (0 or omit to append)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 2 {
						return fmt.Errorf("usage: nib scene add <chapter-index> <slug>")
					}
					chapterIndex, err := strconv.Atoi(args.Get(0))
					if err != nil {
						return fmt.Errorf("invalid chapter index: %s", args.Get(0))
					}
					slug := args.Get(1)
					opts := scene.AddOptions{
						ChapterIndex: chapterIndex,
						Slug:         slug,
						At:           int(cmd.Int("at")),
					}
					result, err := scene.Add(opts)
					if err != nil {
						return err
					}
					fmt.Printf("Added scene %q to chapter %d\n", slug, chapterIndex)

					// Auto-set focus (best-effort)
					projectRoot, _, book, err := bookio.Load()
					if err == nil {
						_, focusErr := scene.SetFocus(projectRoot, book, result.ChapterIndex, result.Position)
						if focusErr == nil {
							fmt.Printf("Focus set to %d.%d\n", result.ChapterIndex, result.Position)
						}
					}
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"li"},
				Usage:   "list scenes by chapter",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "chapter",
						Usage: "1-based chapter index (0 or omit for all)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					opts := scene.ListOptions{
						ChapterIndex: int(cmd.Int("chapter")),
					}
					groups, err := scene.List(opts)
					if err != nil {
						return err
					}
					fmt.Print(scene.FormatList(groups))
					return nil
				},
			},
			{
				Name:      "remove",
				Aliases:   []string{"rm"},
				Usage:     "remove a scene from a chapter",
				ArgsUsage: "<chapter-index> <slug>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 2 {
						return fmt.Errorf("usage: nib scene remove <chapter-index> <slug>")
					}
					chapterIndex, err := strconv.Atoi(args.Get(0))
					if err != nil {
						return fmt.Errorf("invalid chapter index: %s", args.Get(0))
					}
					slug := args.Get(1)
					if err := scene.Remove(scene.RemoveOptions{
						ChapterIndex: chapterIndex,
						Slug:         slug,
					}); err != nil {
						return err
					}
					fmt.Printf("Removed scene %q from chapter %d\n", slug, chapterIndex)
					return nil
				},
			},
			{
				Name:      "edit",
				Aliases:   []string{"ed"},
				Usage:     "open a scene file in your editor",
				ArgsUsage: "[slug]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() > 1 {
						return fmt.Errorf("usage: nib scene edit [slug]")
					}
					var slug string
					if args.Len() == 1 {
						slug = args.First()
					}
					return scene.Edit(scene.EditOptions{Slug: slug})
				},
			},
			{
				Name:      "rename",
				Aliases:   []string{"rn"},
				Usage:     "rename a scene slug",
				ArgsUsage: "<old-slug> <new-slug>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 2 {
						return fmt.Errorf("usage: nib scene rename <old-slug> <new-slug>")
					}
					oldSlug := args.Get(0)
					newSlug := args.Get(1)
					if err := scene.Rename(scene.RenameOptions{
						OldSlug: oldSlug,
						NewSlug: newSlug,
					}); err != nil {
						return err
					}
					fmt.Printf("Renamed scene %q to %q\n", oldSlug, newSlug)
					return nil
				},
			},
			{
				Name:      "move",
				Aliases:   []string{"mo"},
				Usage:     "move a scene using dotted notation (e.g. 3.1 4.2)",
				ArgsUsage: "<from> [to]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() < 1 || args.Len() > 2 {
						return fmt.Errorf("usage: nib scene move <chapter.scene> [chapter[.scene]]")
					}
					var to string
					if args.Len() == 2 {
						to = args.Get(1)
					}
					opts, err := scene.ParseMoveArgs(args.Get(0), to)
					if err != nil {
						return err
					}
					if err := scene.Move(*opts); err != nil {
						return err
					}
					dest := opts.ChapterIndex
					if opts.To != 0 {
						dest = opts.To
					}
					if opts.ToPosition == 0 {
						fmt.Printf("Moved scene to end of chapter %d\n", dest)
					} else {
						fmt.Printf("Moved scene to chapter %d, position %d\n", dest, opts.ToPosition)
					}
					return nil
				},
			},
			{
				Name:      "focus",
				Aliases:   []string{"fo"},
				Usage:     "set or show the current scene focus",
				ArgsUsage: "[chapter[.scene]]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() > 1 {
						return fmt.Errorf("usage: nib scene focus [chapter[.scene]]")
					}

					projectRoot, _, book, err := bookio.Load()
					if err != nil {
						return err
					}

					if args.Len() == 0 {
						// Show current focus
						info, err := scene.GetFocus(projectRoot, book)
						if err != nil {
							return err
						}
						if info == nil {
							fmt.Println("No focus set")
							return nil
						}
						if info.Slug != "" {
							fmt.Printf("Focus: %d.%d (%s)\n", info.Chapter, info.Position, info.Slug)
						} else {
							fmt.Printf("Focus: chapter %d\n", info.Chapter)
						}
						return nil
					}

					// Set focus
					ch, pos, err := scene.ParseDotted(args.First())
					if err != nil {
						return err
					}
					info, err := scene.SetFocus(projectRoot, book, ch, pos)
					if err != nil {
						return err
					}
					if info.Slug != "" {
						fmt.Printf("Focus set to %d.%d (%s)\n", info.Chapter, info.Position, info.Slug)
					} else {
						fmt.Printf("Focus set to chapter %d\n", info.Chapter)
					}
					return nil
				},
			},
			{
				Name:    "unfocus",
				Aliases: []string{"un"},
				Usage:   "clear the current scene focus",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					projectRoot, _, _, err := bookio.Load()
					if err != nil {
						return err
					}
					if err := scene.ClearFocus(projectRoot); err != nil {
						return err
					}
					fmt.Println("Focus cleared")
					return nil
				},
			},
		},
	}
}

func continuityCommand() *cli.Command {
	return &cli.Command{
		Name:    "continuity",
		Aliases: []string{"ct"},
		Usage:   "continuity tracking and analysis",
		Commands: []*cli.Command{
			{
				Name:      "recap",
				Aliases:   []string{"rec"},
				Usage:     "output a JSON recap of chapters from indexed data",
				ArgsUsage: "<range>",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "character",
						Aliases: []string{"c"},
						Usage:   "filter to scenes involving this character (repeatable)",
					},
					&cli.BoolFlag{
						Name:  "detailed",
						Usage: "include facts, locations, dates, times, and mentioned characters",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib continuity recap <range>\n\nExamples:\n  nib ct recap 3                       # recap chapter 3\n  nib ct recap 1-5                     # recap chapters 1 through 5\n  nib ct recap 1,3,5                   # recap chapters 1, 3, and 5\n  nib ct recap 1-5 -c lance-thurgood   # only scenes with lance\n  nib ct recap 1-5 --detailed          # include all indexed data")
					}
					return continuity.Recap(continuity.RecapOptions{
						Range:      args.First(),
						Characters: cmd.StringSlice("character"),
						Detailed:   cmd.Bool("detailed"),
						Pretty:     cmd.Bool("pretty"),
					})
				},
			},
			{
				Name:      "check",
				Aliases:   []string{"ck"},
				Usage:     "check scenes for continuity errors",
				ArgsUsage: "<range>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib continuity check <range>\n\nExamples:\n  nib ct check 3.2       # check a single scene\n  nib ct check 1-3       # check all scenes in chapters 1-3\n  nib ct check 1,3,5     # check all scenes in chapters 1, 3, and 5")
					}
					return continuity.Check(continuity.CheckOptions{
						Range: args.First(),
					})
				},
			},
			{
				Name:      "index",
				Aliases:   []string{"ix"},
				Usage:     "extract structured data from scenes for continuity tracking",
				ArgsUsage: "[range]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "print prompt, command, and raw response from Claude",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "re-index even if scene file hasn't changed",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() > 1 {
						return fmt.Errorf("usage: nib continuity index [range]\n\nExamples:\n  nib ct index 3.2       # single scene\n  nib ct index 1-3       # all scenes in chapters 1-3\n  nib ct index 1.1-2.3   # chapter 1 scene 1 through chapter 2 scene 3\n  nib ct index 1,3,5     # all scenes in chapters 1, 3, and 5\n  nib ct index 2.1,2.3   # specific scenes by dotted notation")
					}
					var rangeArg string
					if args.Len() == 1 {
						rangeArg = args.First()
					}
					return continuity.Index(continuity.IndexOptions{
						Range:   rangeArg,
						Verbose: cmd.Bool("verbose"),
						Force:   cmd.Bool("force"),
					})
				},
			},
			{
				Name:      "characters",
				Aliases:   []string{"chr"},
				Usage:     "list characters from indexed data as sorted, deduplicated JSON (pov+present only; use --all to include mentioned)",
				ArgsUsage: "[range]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "all",
						Usage: "include mentioned characters (default: pov and present only)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() > 1 {
						return fmt.Errorf("usage: nib ct characters [range]\n\nExamples:\n  nib ct characters              # all characters\n  nib ct characters 3            # characters in chapter 3\n  nib ct characters 1-5          # characters in chapters 1-5\n  nib ct characters 2.1          # characters in chapter 2 scene 1\n  nib ct characters --all 1,3    # include mentioned characters")
					}
					var rangeArg string
					if args.Len() == 1 {
						rangeArg = args.First()
					}
					return continuity.Characters(continuity.CharactersOptions{
						Range:  rangeArg,
						All:    cmd.Bool("all"),
						Pretty: cmd.Bool("pretty"),
					})
				},
			},
			{
				Name:      "chapters",
				Aliases:   []string{"chp"},
				Usage:     "list scenes where characters appear (pov/present) in dotted notation",
				ArgsUsage: "<character> [character...]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "and",
						Usage: "scenes where ALL characters appear (default)",
					},
					&cli.BoolFlag{
						Name:  "or",
						Usage: "scenes where ANY character appears, grouped by character",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() == 0 {
						return fmt.Errorf("usage: nib ct chapters <character> [character...]\n\nExamples:\n  nib ct chapters lance bo          # scenes with both (AND)\n  nib ct chapters --or lance bo     # scenes per character (OR)")
					}
					if cmd.Bool("and") && cmd.Bool("or") {
						return fmt.Errorf("--and and --or are mutually exclusive")
					}
					return continuity.Chapters(continuity.ChaptersOptions{
						Characters: args.Slice(),
						Or:         cmd.Bool("or"),
						Pretty:     cmd.Bool("pretty"),
					})
				},
			},
			{
				Name:      "ask",
				Aliases:   []string{"as"},
				Usage:     "ask a plain-English question about the novel",
				ArgsUsage: "<question>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "range",
						Usage: "limit search to a chapter/scene range (e.g. 1-10, 3.1-5.2)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() == 0 {
						return fmt.Errorf("usage: nib ct ask \"your question here\"")
					}
					question := strings.Join(args.Slice(), " ")
					return continuity.Ask(continuity.AskOptions{
						Question: question,
						Range:    cmd.String("range"),
					})
				},
			},
			{
				Name:    "reset",
				Aliases: []string{"res"},
				Usage:   "clear all indexed continuity data",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "yes",
						Usage: "skip confirmation prompt",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return continuity.Reset(continuity.ResetOptions{
						Yes: cmd.Bool("yes"),
					})
				},
			},
		},
	}
}

func manuscriptCommand() *cli.Command {
	return &cli.Command{
		Name:    "manuscript",
		Aliases: []string{"ma"},
		Usage:   "manuscript assembly and status",
		Commands: []*cli.Command{
			{
				Name:      "build",
				Aliases:   []string{"bu"},
				Usage:     "assemble manuscript and build output format",
				ArgsUsage: "[format]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "scene-headings",
						Usage: "include scene filenames as headings in the output",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					formatStr := "md"
					if cmd.Args().Len() > 0 {
						formatStr = cmd.Args().First()
					}
					format, err := manuscript.ParseFormat(formatStr)
					if err != nil {
						return err
					}
					return manuscript.Build(format, nil, cmd.Bool("scene-headings"))
				},
			},
			{
				Name:    "status",
				Aliases: []string{"st"},
				Usage:   "show manuscript statistics",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					status, err := manuscript.GetStatus()
					if err != nil {
						return err
					}
					fmt.Print(manuscript.FormatStatus(status))
					return nil
				},
			},
			{
				Name:    "toc",
				Aliases: []string{"to"},
				Usage:   "show table of contents with dotted notation",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return manuscript.TOC(os.Stdout)
				},
			},
			{
				Name:      "search",
				Aliases:   []string{"se"},
				Usage:     "search scenes with a plain-English query (e.g. 1-3, 1.1-2.3, 1,2,4)",
				ArgsUsage: "<range> <query>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() < 2 {
						return fmt.Errorf("usage: nib manuscript search <range> \"<query>\"\n\nExamples:\n  nib ma search 1-41 \"body words near emotion verbs\"\n  nib ma se 1.1-5.3 \"dialogue where characters lie\"")
					}
					query := strings.Join(args.Slice()[1:], " ")
					return manuscript.Search(manuscript.SearchOptions{
						Range: args.First(),
						Query: query,
					})
				},
			},
			{
				Name:      "critique",
				Aliases:   []string{"cr"},
				Usage:     "review scenes with Claude Code (e.g. 1-3, 1.1-2.3, 1,2,4)",
				ArgsUsage: "<range>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib manuscript critique <range>\n\nExamples:\n  nib ma critique 1-3       # all scenes in chapters 1-3\n  nib ma critique 1.1-2.3   # chapter 1 scene 1 through chapter 2 scene 3\n  nib ma critique 1,3,5     # all scenes in chapters 1, 3, and 5\n  nib ma critique 2.1,2.3   # specific scenes by dotted notation")
					}
					return manuscript.Critique(manuscript.CritiqueOptions{
						Range: args.First(),
					})
				},
			},
			{
				Name:      "proof",
				Aliases:   []string{"pr"},
				Usage:     "copy-edit scenes with Claude Code (e.g. 1-3, 1.1-2.3, 1,2,4)",
				ArgsUsage: "<range>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() != 1 {
						return fmt.Errorf("usage: nib manuscript proof <range>\n\nExamples:\n  nib ma proof 1-3       # all scenes in chapters 1-3\n  nib ma proof 1.1-2.3   # chapter 1 scene 1 through chapter 2 scene 3\n  nib ma proof 1,3,5     # all scenes in chapters 1, 3, and 5\n  nib ma proof 2.1,2.3   # specific scenes by dotted notation")
					}
					return manuscript.Proof(manuscript.ProofOptions{
						Range: args.First(),
					})
				},
			},
			{
				Name:      "voice",
				Aliases:   []string{"vo"},
				Usage:     "check character voice consistency across the manuscript",
				ArgsUsage: "<character> [character...]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "thorough",
						Usage: "sample 60% of scenes instead of 30% for higher confidence",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() == 0 {
						return fmt.Errorf("usage: nib manuscript voice [--thorough] <character> [character...]\n\nExamples:\n  nib ma voice lance-thurgood\n  nib ma vo --thorough lance-thurgood bo-dupuis")
					}
					return manuscript.Voice(manuscript.VoiceOptions{
						Characters: args.Slice(),
						Thorough:   cmd.Bool("thorough"),
					})
				},
			},
		},
	}
}
