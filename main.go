// Copyright 2018 Jacques Supcik
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This program fetches the level of some lakes in the canton of Fribourg
// and makes them available for simple web sites of for IoT.
// It stores the lake levels in a Firebase Realtime Database

package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go"
	"github.com/PuerkitoBio/goquery"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	pageURL = "https://www.groupe-e.ch/fr/univers-groupe-e/niveau-lacs"
)

// Lake is the structure for storing lake information.
type Lake struct {
	Name      string    `datastore:"name"`
	MaxLevel  float64   `datastore:"max_level,noindex"`
	Date      time.Time `datastore:"date,noindex"`
	Today     float64   `datastore:"today,noindex"`
	Yesterday float64   `datastore:"yesterday,noindex"`
}

// Lakes is the list of all fetched lakes.
type Lakes map[string]Lake

// msm parses a string representing a lake's level ang returns a float
// note: "msm" means "mètres sur mer" (in french) which means "metres above sea level"
func msm(t string) float64 {
	re := regexp.MustCompile(`(\d+\.\d+).*msm`)
	n := re.FindStringSubmatch(t)
	if n != nil {
		nf, err := strconv.ParseFloat(n[1], 64)
		if err == nil {
			return nf
		}
	}
	return 0
}

// scrape reads the web page from "Groupe E" and extracts relevant information
// for lake level.
func scrape(r io.Reader) (Lakes, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	result := make(Lakes)
	table := doc.Find("table").First()
	header := table.Find("thead tr th")
	date1, err := time.Parse("2.1.2006", strings.TrimSpace(header.Eq(2).Text()))
	if err != nil {
		return nil, err
	}
	date2, err := time.Parse("2.1.2006", strings.TrimSpace(header.Eq(3).Text()))
	if err != nil {
		return nil, err
	}
	body := table.Find("tbody tr")
	body.Each(func(i int, selection *goquery.Selection) {
		name := strings.TrimSpace(selection.Find("td").Eq(0).Text())
		maxLevel := msm(strings.TrimSpace(selection.Find("td").Eq(1).Text()))
		l1 := msm(strings.TrimSpace(selection.Find("td").Eq(2).Text()))
		l2 := msm(strings.TrimSpace(selection.Find("td").Eq(3).Text()))
		lake := Lake{
			Name:     name,
			MaxLevel: maxLevel,
		}
		if date1.After(date2) {
			lake.Date = date1
			lake.Today = l1
			lake.Yesterday = l2
		} else {
			lake.Date = date2
			lake.Today = l2
			lake.Yesterday = l1
		}
		result[name] = lake
	})
	return result, nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	resp, err := urlfetch.Client(ctx).Get(pageURL)
	if err != nil {
		log.Errorf(ctx, "Error fetching URL : %v", err)
		http.Error(w, "Application Server Error", 500)
		return
	}
	lakes, err := scrape(resp.Body)
	if err != nil {
		log.Errorf(ctx, "Error scraping data : %v", err)
		http.Error(w, "Application Server Error", 500)
		return
	}

	fbConfig := &firebase.Config{
		DatabaseURL: "https://niveau-lacs.firebaseio.com/",
	}
	app, err := firebase.NewApp(ctx, fbConfig)
	if err != nil {
		log.Errorf(ctx, "Error creating firebase app: %v", err)
		http.Error(w, "Application Server Error", 500)
		return
	}

	dbClient, err := app.Database(ctx)
	if err != nil {
		log.Errorf(ctx, "Error connecting to database : %v", err)
		http.Error(w, "Application Server Error", 500)
		return
	}

	log.Infof(ctx, "Lakes: %v", lakes)
	for name, l := range lakes {
		ref := dbClient.NewRef("/current/" + name)
		err = ref.Set(ctx, &l)
		if err != nil {
			log.Errorf(ctx, "Error writing datastore : %v", err)
			http.Error(w, "Application Server Error", 500)
			return
		}
	}
	fmt.Fprintln(w, "Done") // nolint: gas
}

func main() {
	http.HandleFunc("/", handle)
	appengine.Main()
}
