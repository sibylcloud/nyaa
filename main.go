package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"html"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var db *gorm.DB
var templates = template.Must(template.ParseFiles("index.html", "FAQ.html"))
var debugLogger *log.Logger
var trackers = "&tr=udp://zer0day.to:1337/announce&tr=udp://tracker.leechers-paradise.org:6969&tr=udp://explodie.org:6969&tr=udp://tracker.opentrackr.org:1337&tr=udp://tracker.coppersurfer.tk:6969"

func getDBHandle() *gorm.DB {
	dbInit, err := gorm.Open("sqlite3", "./nyaa.db")

	// Migrate the schema of Torrents
	// dbInit.AutoMigrate(&Torrents{})
	// dbInit.AutoMigrate(&SubCategories{})

	checkErr(err)
	return dbInit
}

func checkErr(err error) {
	if err != nil {
		debugLogger.Println("   " + err.Error())
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	page := vars["page"]
	pagenum, _ := strconv.Atoi(html.EscapeString(page))

	b := CategoryJson{Torrents: []TorrentsJson{}}
	maxPerPage := 50
	nbTorrents := 0

	torrents := getAllTorrents(maxPerPage, maxPerPage*(pagenum-1))
	for i, _ := range torrents {
		nbTorrents++
		res := torrents[i].toJson()

		b.Torrents = append(b.Torrents, res)
	}
	b.QueryRecordCount = maxPerPage
	b.TotalRecordCount = nbTorrents
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(b)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func singleapiHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]
	b := CategoryJson{Torrents: []TorrentsJson{}}

	torrent, err := getTorrentById(id)
	res := torrent.toJson()
	b.Torrents = append(b.Torrents, res)

	b.QueryRecordCount = 1
	b.TotalRecordCount = 1
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(b)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page := vars["page"]

	// db params url
	maxPerPage, errConv := strconv.Atoi(r.URL.Query().Get("max"))
	if errConv != nil {
		maxPerPage = 50 // default Value maxPerPage
	}
	pagenum, _ := strconv.Atoi(html.EscapeString(page))
	searchQuery := r.URL.Query().Get("q")
	cat := r.URL.Query().Get("c")
	searchCatId := html.EscapeString(strings.Split(cat, "_")[0])
	searchSubCatId := html.EscapeString(strings.Split(cat, "_")[1])

	nbTorrents := 0

	b := []TorrentsJson{}

	torrents := getTorrents(createWhereParams("torrent_name LIKE ? AND category_id LIKE ? AND sub_category_id LIKE ?", "%"+searchQuery+"%", searchCatId+"%", searchSubCatId+"%"), maxPerPage, maxPerPage*(pagenum-1))

	for i, _ := range torrents {
		nbTorrents++
		res := torrents[i].toJson()

		b = append(b, res)

	}

	htv := HomeTemplateVariables{b, getAllCategories(false), searchQuery, cat, maxPerPage, nbTorrents}

	err := templates.ExecuteTemplate(w, "index.html", htv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func safe(s string) template.URL {
	return template.URL(s)
}

func faqHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "FAQ.html", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page := vars["page"]

	// db params url
	maxPerPage, errConv := strconv.Atoi(r.URL.Query().Get("max"))
	if errConv != nil {
		maxPerPage = 50 // default Value maxPerPage
	}

	nbTorrents := 0
	pagenum, _ := strconv.Atoi(html.EscapeString(page))
	b := []TorrentsJson{}
	torrents := getAllTorrents(maxPerPage, maxPerPage*(pagenum-1))
	for i, _ := range torrents {
		nbTorrents++
		res := torrents[i].toJson()

		b = append(b, res)

	}

	htv := HomeTemplateVariables{b, getAllCategories(false), "", "_", maxPerPage, nbTorrents}

	err := templates.ExecuteTemplate(w, "index.html", htv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func main() {

	db = getDBHandle()
	router := mux.NewRouter()

	cssHandler := http.FileServer(http.Dir("./css/"))
	jsHandler := http.FileServer(http.Dir("./js/"))
	http.Handle("/css/", http.StripPrefix("/css/", cssHandler))
	http.Handle("/js/", http.StripPrefix("/js/", jsHandler))

	// Routes,
	router.HandleFunc("/", rootHandler)
	router.HandleFunc("/page/{page}", rootHandler)
	router.HandleFunc("/search", searchHandler)
	router.HandleFunc("/search/{page}", searchHandler)
	router.HandleFunc("/api/{page}", apiHandler).Methods("GET")
	router.HandleFunc("/api/torrent/{id}", singleapiHandler).Methods("GET")
	router.HandleFunc("/faq", faqHandler)

	http.Handle("/", router)

	// Set up server,
	srv := &http.Server{
		Addr:         "localhost:9999",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	err := srv.ListenAndServe()
	checkErr(err)
}
