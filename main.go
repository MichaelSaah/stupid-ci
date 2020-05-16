package main

import (
	"encoding/json"
	"fmt"
	"log"
	//"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mediocregopher/radix/v3"
)

var client radix.Client

type SubmittedJob struct {
	ResourcePath string `json:"resource_path"`
}

type InternalJob struct {
	ResourcePath string `json:"resource_path"`
	UUID string `json:"uuid"`
	SubmittedAt int64 `json:"submitted_at"`
	// SubmittedBy
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to stupid-ci!")
}

func createJob(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var job SubmittedJob

	err := decoder.Decode(&job)
	if err != nil {
		panic(err)
	}

	internalJob := InternalJob{
		ResourcePath: job.ResourcePath,
		UUID: uuid.New().String(),
		SubmittedAt: time.Now().Unix(),
	}

	internalJobJson, err := json.Marshal(internalJob)
	if err != nil {
		panic(err)
	}

	// record job details
	client.Do(radix.Cmd(nil, "SET", internalJob.UUID, string(internalJobJson)))

	// add job to queue
	client.Do(radix.Cmd(nil, "LPUSH", "jobs", internalJob.UUID))

	log.Println(fmt.Sprintf("Job submitted: %s", job.ResourcePath))
}

func checkContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("content-type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("{\"error\": \"we only speak json here\"}"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)

	router.Use(setContentType)
	router.Use(checkContentType)

	router.HandleFunc("/", homePage)
	router.HandleFunc("/jobs", createJob).Methods("POST")
	//router.HandleFunc("/jobs", getJobs).Methods("GET")
	//router.HandleFunc("/jobs/{uuid}", getJob).Methods("GET")
	//router.HandleFunc("/jobs/{uuid}", deleteJob).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8888", router))
}

func main() {
	// init redis
	var err error
	client, err = radix.NewPool("tcp", "127.0.0.1:6379", 8)
	if err != nil {
		panic(err)
	}

	handleRequests()
}
