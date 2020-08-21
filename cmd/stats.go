package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/applandinc/appland-cli/internal/config"
	"github.com/applandinc/appland-cli/internal/files"
	"github.com/spf13/cobra"
)

type stat struct {
	Key   string
	Value int
}

func sortStats(stats map[string]int) []stat {
	var s []stat
	for k, v := range stats {
		s = append(s, stat{k, v})
	}

	sort.Slice(s, func(i, j int) bool {
		return s[i].Value > s[j].Value
	})

	return s
}

func init() {
	var (
		statsCmd = &cobra.Command{
			Use:   "stats",
			Short: "Show statistics for AppMaps",
			RunE: func(cmd *cobra.Command, args []string) error {
				validator := func(fi os.FileInfo) bool {
					if fi.Size() > 1024*1024*2000 {
						warn(fmt.Errorf(">>> File %s too big, skipped", fi.Name()))
						return false
					}
					return true
				}

				appmaps, err := files.FindAppMaps(args, validator)
				if err != nil {
					return fmt.Errorf("Failed finding AppMaps: %w", err)
				}

				var totalCalls uint64 = 0
				globalCounts := make(map[string]int)
				fs := config.GetFS()
				fmt.Fprintf(os.Stderr, "Found %d appmaps\n", len(appmaps))
				for _, appmap := range appmaps {
					f, err := fs.Open(appmap)
					if err != nil {
						return fmt.Errorf("Failed opening %s: %w", appmap, err)
					}
					dec := json.NewDecoder(f)
					var v map[string]interface{}
					err = dec.Decode(&v)
					f.Close()
					if err != nil {
						warn(fmt.Errorf(">>> Failed decoding %s, %w", appmap, err))
						continue
					}
					if v["events"] == nil {
						warn(fmt.Errorf(">>> events is nil in %s", appmap))
						continue
					}
					events := v["events"].([]interface{})
					counts := make(map[string]int)
					fmt.Printf("%s: %d event(s)\n", appmap, len(events))
					for _, event := range events {
						e := event.(map[string]interface{})
						if e["event"] != "call" {
							continue
						}
						c := e["defined_class"].(string)
						m := e["method_id"].(string)
						sep := "#"
						if e["static"].(bool) {
							sep = "."
						}
						id := strings.Join([]string{c, sep, m}, "")
						counts[id]++
						globalCounts[id]++
						totalCalls++
					}

					stats := sortStats(counts)
					max := 20
					if len(stats) < max {
						max = len(stats)
					}
					fmt.Printf("Top %d\n", max)
					for i := 0; i < max; i++ {
						fmt.Printf("%s: %d\n", stats[i].Key, stats[i].Value)
						if stats[i].Value == 1 {
							break
						}
					}
				}

				stats := sortStats(globalCounts)
				fmt.Printf("Total calls: %v\n", totalCalls)
				max := 20
				if len(stats) < max {
					max = len(stats)
				}
				fmt.Printf("Top %d:\n", max)
				for i := 0; i < max; i++ {
					fmt.Printf("%s: %d\n", stats[i].Key, stats[i].Value)
				}

				return nil
			},
		}
	)
	rootCmd.AddCommand(statsCmd)
}
