package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"github.com/fatih/color"
	"github.com/renstrom/fuzzysearch/fuzzy"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const achievements = "https://raw.githubusercontent.com/xivapi/ffxiv-datamining/master/csv/Achievement.csv"

var patches = map[string]string{
	"alexander: the creator": "alexander",
	"alexander: the creator (savage)": "alexander (savage)",
	"minstrel's ballad: the weapon's refrain (ultimate)": "weapon's refrain (ultimate)",
}

func main() {
	log.Printf("Fetching %s...", achievements)
	achCsvResponse, err := http.Get(achievements)
	if err != nil {
		log.Fatal(err)
	}
	if achCsvResponse.StatusCode != 200 {
		log.Fatal(achCsvResponse.Status)
	}

	achCsv := csv.NewReader(achCsvResponse.Body)
	csvlines, err := achCsv.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	recs := records{}

	for _, line := range csvlines {
		if len(line[3]) > 0 && len(line[0]) > 0 {
			id, _ := strconv.Atoi(line[0])
			recs.ids = append(recs.ids, id)
			recs.names = append(recs.names, line[2])
			recs.descriptions = append(recs.descriptions, line[3])
		}
	}

	log.Printf("Got %d achievements", len(recs.ids))

	log.Println("Reading stdin...")

	lineIn := bufio.NewReader(os.Stdin)

	achMap := map[string]int{}

	for {
		bLine, err := lineIn.ReadBytes('\n')
		if err != nil {
			break
		}

		duty := regexp.MustCompile(`^\s*the\s*`).ReplaceAllString(strings.ToLower(strings.TrimSpace(string(bLine))), "")
		if patched := patches[duty]; patched != "" {
			duty = patched
		}

		if len(duty) < 3 {
			continue
		}

		id := recs.findBest(duty)
		if id != -1 {
			achMap[duty] = id
		}
	}

	je := json.NewEncoder(os.Stdout)
	je.SetIndent("", "  ")
	je.Encode(achMap)
}

type records struct {
	ids          []int
	names        []string
	descriptions []string
}

func (r *records) findBest(duty string) int {
	if results := fuzzy.RankFindFold("mapping " + duty, r.names); len(results) > 0 {
		sort.Sort(results)
		return r.ids[results[0].OriginalIndex]
	} else if results := fuzzy.RankFindFold(duty, r.names); len(results) > 0 {
		sort.Sort(results)
		return r.ids[results[0].OriginalIndex]
	} else if results := fuzzy.RankFindFold(duty, r.descriptions); len(results) > 0 {
		log.Printf(color.YellowString("Falling back to description match for %s", duty))
		sort.Sort(results)
		return r.ids[results[0].OriginalIndex]
	} else {
		log.Println(color.RedString("Could not find an achievement for %s", duty))
	}

	return -1
}
