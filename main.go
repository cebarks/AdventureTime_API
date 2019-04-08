package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	db "github.com/SiCuellar/AdventureTime_API/migrations"
)

func main() {
	db.Migrate()

	db.Connect()

	router := mux.NewRouter()

	router.HandleFunc("/api/v1/login", LoginHandler).Methods("POST")
	router.HandleFunc("/api/v1/quest", QuestHandler).Methods("POST")

	fmt.Println("Listening on port: " + os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	user := db.Connection.Preload("Items").Find(&db.User{}, params["user_id"])
	json.NewEncoder(w).Encode(user)
}

func QuestHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	userID, _ := strconv.ParseUint(params["user_id"][0], 10, 32)
	lat := params["lat"][0]
	long := params["long"][0]

	var oldQuest db.Quest

	query := db.Connection.Where("status = ?", 0).Where("user_id = ?", userID).First(&oldQuest)

	var quest db.Quest

	if query.RecordNotFound() {
		fmt.Println("Previous Quest not found. Generating new quest.")
		quest = buildQuest(lat, long, userID)
	} else {
		fmt.Println("Previous Quest Found! ")
		quest = oldQuest
	}

	json.NewEncoder(w).Encode(quest)
}

func getQuestLocations(lat string, long string) []FsItem {
	resp, err := http.Get("https://api.foursquare.com/v2/venues/explore?client_id=" + os.Getenv("FOUR_ID") + "&client_secret=" + os.Getenv("FOUR_SECRET") + "&v=20190404&ll=" + lat + "," + long + ",&radius=750&limit=4")

	if err != nil {
		log.Fatal(err)
	} else {
		data, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		var result Result
		json.Unmarshal([]byte(data), &result)
		return result.Response.Groups[0].Items
	}

	return []FsItem{}
}

func buildQuest(lat string, long string, userID uint64) db.Quest {
	locations := getQuestLocations(lat, long)

	quest := db.Quest{
		Location1: locations[0].Venue.Id + "|" + strings.Join(locations[0].Venue.Location.FormattedAddress, ", "),
		Location2: locations[1].Venue.Id + "|" + strings.Join(locations[1].Venue.Location.FormattedAddress, ", "),
		Location3: locations[2].Venue.Id + "|" + strings.Join(locations[2].Venue.Location.FormattedAddress, ", "),
		UserID:    uint(userID)}

	db.NewQuest(quest)

	return quest
}

type Result struct {
	Response struct {
		Groups []struct {
			Items []FsItem
		}
	}
}

type Location struct {
	Lat              float64
	Lng              float64
	Distance         int
	FormattedAddress []string // check this later
}

type Venue struct {
	Id       string
	Name     string
	Location Location
}
type FsItem struct {
	Venue Venue
}
